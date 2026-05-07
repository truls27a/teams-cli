package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	watchInterval time.Duration
	watchNotifier string
)

type watchEntry struct {
	LastMessageID string `json:"lastMessageId"`
	LastArrival   string `json:"lastArrival"`
}

type watchState struct {
	Chats map[string]watchEntry `json:"chats"`
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch chats and fire notifications on new messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}

		var n notifier
		switch watchNotifier {
		case "auto":
			if runtime.GOOS == "darwin" {
				n = macNotifier{}
			} else {
				n = stderrNotifier{}
			}
		case "mac":
			n = macNotifier{}
		case "stderr":
			n = stderrNotifier{}
		default:
			return fmt.Errorf("unknown notifier %q (auto|mac|stderr)", watchNotifier)
		}

		home, _ := os.UserHomeDir()
		statePath := filepath.Join(home, ".config", "teams", "watch-state.json")

		state := &watchState{Chats: map[string]watchEntry{}}
		seeded := false
		if b, err := os.ReadFile(statePath); err == nil {
			if err := json.Unmarshal(b, state); err != nil {
				return err
			}
			if state.Chats == nil {
				state.Chats = map[string]watchEntry{}
			}
			seeded = true
		} else if !os.IsNotExist(err) {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		tick := func() {
			chats, err := client.ListChats(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "watch tick: %v\n", err)
				return
			}
			names := resolveMissingNames(ctx, client, chats)

			for _, c := range chats {
				if c.IsEmptyConversation || c.LastMessage == nil || strings.HasPrefix(c.ID, "19:meeting_") {
					continue
				}
				prev, known := state.Chats[c.ID]
				if !seeded || !known {
					state.Chats[c.ID] = watchEntry{c.LastMessage.ID, c.LastMessage.OriginalArrivalTime}
					continue
				}
				if c.LastMessage.ID == prev.LastMessageID {
					continue
				}

				const cap = 50
				msgs, _, err := client.ListMessages(ctx, c.ID, cap)
				if err != nil {
					fmt.Fprintf(os.Stderr, "watch %s: %v\n", c.ID, err)
					continue
				}
				sort.SliceStable(msgs, func(i, j int) bool {
					return messageTime(msgs[i]).Before(messageTime(msgs[j]))
				})
				idx := -1
				for i, m := range msgs {
					if m.ID == prev.LastMessageID {
						idx = i
						break
					}
				}

				threadName := chatDisplay(c, client.SelfMRI, names)
				isGroup := chatType(c) == "group"

				fire := func(sender, content, messagetype, suffix string) {
					if sender == "" {
						sender = "Someone"
					}
					body := renderContent(content, messagetype, nil)
					body = strings.ReplaceAll(body, "*", "")
					body = strings.ReplaceAll(body, "_", "")
					body = strings.Join(strings.Fields(body), " ")
					if len(body) > 140 {
						body = body[:139] + "…"
					}
					title := sender + suffix
					if isGroup {
						title = threadName + suffix
						body = sender + ": " + body
					}
					if err := n.notify(title, "Teams CLI", body); err != nil {
						fmt.Fprintf(os.Stderr, "notify: %v\n", err)
					}
				}

				if idx < 0 && len(msgs) >= cap {
					fire(c.LastMessage.IMDisplayName, c.LastMessage.Content, c.LastMessage.MessageType, " (50+ new)")
				} else {
					for _, m := range msgs[idx+1:] {
						if m.From == client.SelfMRI || strings.HasSuffix(m.From, "/"+client.SelfMRI) {
							continue
						}
						if strings.HasPrefix(m.Messagetype, "Control/") || strings.HasPrefix(m.Messagetype, "ThreadActivity/") {
							continue
						}
						fire(m.IMDisplayName, m.Content, m.Messagetype, "")
					}
				}

				state.Chats[c.ID] = watchEntry{c.LastMessage.ID, c.LastMessage.OriginalArrivalTime}
			}

			if err := os.MkdirAll(filepath.Dir(statePath), 0700); err == nil {
				b, _ := json.MarshalIndent(state, "", "  ")
				_ = os.WriteFile(statePath, b, 0600)
			}
		}

		tick()
		seeded = true
		t := time.NewTicker(watchInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				tick()
			}
		}
	},
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 20*time.Second, "polling interval")
	watchCmd.Flags().StringVar(&watchNotifier, "notifier", "auto", "notifier backend (auto|mac|stderr)")
}

type notifier interface {
	notify(title, subtitle, body string) error
}

type macNotifier struct{}

func (macNotifier) notify(title, subtitle, body string) error {
	esc := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		return strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' {
				return ' '
			}
			return r
		}, s)
	}
	script := fmt.Sprintf(`display notification "%s" with title "%s" subtitle "%s" sound name "Ping"`,
		esc(body), esc(title), esc(subtitle))
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type stderrNotifier struct{}

func (stderrNotifier) notify(title, subtitle, body string) error {
	_, err := fmt.Fprintf(os.Stderr, "[%s] %s — %s\n", subtitle, title, body)
	return err
}
