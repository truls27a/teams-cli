package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"teams/teams"
)

var (
	watchInterval time.Duration
	watchNotifier string
	watchOnce     bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch chats and fire notifications on new messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}

		n, err := pickNotifier(watchNotifier)
		if err != nil {
			return err
		}

		state, seeded, err := loadWatchState()
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		tick := func() error {
			return watchTick(ctx, client, n, state, seeded)
		}

		if err := tick(); err != nil {
			fmt.Fprintf(os.Stderr, "watch tick: %v\n", err)
		}
		seeded = true
		if err := saveWatchState(state); err != nil {
			fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
		}

		if watchOnce {
			return nil
		}

		t := time.NewTicker(watchInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				if err := tick(); err != nil {
					fmt.Fprintf(os.Stderr, "watch tick: %v\n", err)
				}
				if err := saveWatchState(state); err != nil {
					fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
				}
			}
		}
	},
}

type watchEntry struct {
	LastMessageID string `json:"lastMessageId"`
	LastArrival   string `json:"lastArrival"`
}

type watchState struct {
	Chats map[string]watchEntry `json:"chats"`
}

func watchStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "teams", "watch-state.json")
}

func loadWatchState() (*watchState, bool, error) {
	b, err := os.ReadFile(watchStatePath())
	if os.IsNotExist(err) {
		return &watchState{Chats: map[string]watchEntry{}}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var s watchState
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, false, err
	}
	if s.Chats == nil {
		s.Chats = map[string]watchEntry{}
	}
	return &s, true, nil
}

func saveWatchState(s *watchState) error {
	path := watchStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path, b, 0600)
}

func watchTick(ctx context.Context, client *teams.Client, n notifier, state *watchState, seeded bool) error {
	chats, err := client.ListChats(ctx)
	if err != nil {
		return err
	}

	names := resolveMissingNames(ctx, client, chats)

	for _, c := range chats {
		if c.IsEmptyConversation || c.LastMessage == nil {
			continue
		}
		if strings.HasPrefix(c.ID, "19:meeting_") {
			continue
		}
		prev, known := state.Chats[c.ID]
		if !seeded || !known {
			state.Chats[c.ID] = watchEntry{
				LastMessageID: c.LastMessage.ID,
				LastArrival:   c.LastMessage.OriginalArrivalTime,
			}
			continue
		}
		if c.LastMessage.ID == prev.LastMessageID {
			continue
		}

		threadName := chatDisplay(c, client.SelfMRI, names)
		isGroup := chatType(c) == "group"
		newMsgs, overflow, err := fetchNewMessages(ctx, client, c.ID, prev.LastMessageID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "watch %s: %v\n", c.ID, err)
			continue
		}

		fire := func(sender, content, messagetype string, suffix string) {
			if sender == "" {
				sender = "Someone"
			}
			var title, body string
			if isGroup {
				title = threadName + suffix
				body = sender + ": " + plainBody(content, messagetype)
			} else {
				title = sender + suffix
				body = plainBody(content, messagetype)
			}
			if err := n.notify(title, "Teams CLI", body); err != nil {
				fmt.Fprintf(os.Stderr, "notify: %v\n", err)
			}
		}

		if overflow {
			fire(c.LastMessage.IMDisplayName, c.LastMessage.Content, c.LastMessage.MessageType, " (50+ new)")
		} else {
			for _, m := range newMsgs {
				if isSelf(m.From, client.SelfMRI) {
					continue
				}
				if isControlMessage(m.Messagetype) {
					continue
				}
				fire(m.IMDisplayName, m.Content, m.Messagetype, "")
			}
		}

		state.Chats[c.ID] = watchEntry{
			LastMessageID: c.LastMessage.ID,
			LastArrival:   c.LastMessage.OriginalArrivalTime,
		}
	}
	return nil
}

func fetchNewMessages(ctx context.Context, client *teams.Client, convID, lastSeenID string) ([]teams.Message, bool, error) {
	const cap = 50
	msgs, _, err := client.ListMessages(ctx, convID, cap)
	if err != nil {
		return nil, false, err
	}
	sort.SliceStable(msgs, func(i, j int) bool {
		return messageTime(msgs[i]).Before(messageTime(msgs[j]))
	})
	idx := -1
	for i, m := range msgs {
		if m.ID == lastSeenID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, len(msgs) >= cap, nil
	}
	return msgs[idx+1:], false, nil
}

func isSelf(from, selfMRI string) bool {
	if from == "" || selfMRI == "" {
		return false
	}
	return from == selfMRI || strings.HasSuffix(from, "/"+selfMRI) || strings.HasSuffix(from, selfMRI)
}

func isControlMessage(mt string) bool {
	return strings.HasPrefix(mt, "Control/") || strings.HasPrefix(mt, "ThreadActivity/")
}

var markdownStripRE = regexp.MustCompile(`\*\*|__|\*|_`)

func plainBody(content, messagetype string) string {
	s := renderContent(content, messagetype, nil)
	s = markdownStripRE.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	const maxLen = 140
	if len(s) > maxLen {
		s = s[:maxLen-1] + "…"
	}
	return s
}

type notifier interface {
	notify(title, subtitle, body string) error
}

func pickNotifier(name string) (notifier, error) {
	switch name {
	case "auto":
		if runtime.GOOS == "darwin" {
			return macNotifier{}, nil
		}
		return stderrNotifier{}, nil
	case "mac":
		return macNotifier{}, nil
	case "stderr":
		return stderrNotifier{}, nil
	}
	return nil, fmt.Errorf("unknown notifier %q (auto|mac|stderr)", name)
}

type macNotifier struct{}

func (macNotifier) notify(title, subtitle, body string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s" subtitle "%s" sound name "Ping"`,
		applescriptEscape(body), applescriptEscape(title), applescriptEscape(subtitle))
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func applescriptEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

type stderrNotifier struct{}

func (stderrNotifier) notify(title, subtitle, body string) error {
	_, err := fmt.Fprintf(os.Stderr, "[%s] %s — %s\n", subtitle, title, body)
	return err
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 20*time.Second, "polling interval")
	watchCmd.Flags().StringVar(&watchNotifier, "notifier", "auto", "notifier backend (auto|mac|stderr)")
	watchCmd.Flags().BoolVar(&watchOnce, "once", false, "run a single tick and exit")
}
