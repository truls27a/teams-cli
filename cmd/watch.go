package cmd

import (
	"context"
	"encoding/json"
	"errors"
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
	"teams/teams"
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
		switch {
		case jsonOutput:
			n = jsonNotifier{enc: json.NewEncoder(os.Stdout)}
		case watchNotifier == "auto":
			if runtime.GOOS == "darwin" {
				n = macNotifier{}
			} else {
				n = stderrNotifier{}
			}
		case watchNotifier == "mac":
			n = macNotifier{}
		case watchNotifier == "stderr":
			n = stderrNotifier{}
		default:
			return fmt.Errorf("unknown notifier %q (auto|mac|stderr)", watchNotifier)
		}

		cache, err := os.UserCacheDir()
		if err != nil {
			home, _ := os.UserHomeDir()
			cache = filepath.Join(home, ".cache")
		}
		statePath := filepath.Join(cache, "teams-cli", "chat-seen.json")

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

		tick := func() error {
			chats, err := client.ListChats(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "watch tick: %v\n", err)
				var apiErr *teams.APIError
				if errors.As(err, &apiErr) && (apiErr.Status == 401 || apiErr.Status == 403) {
					nc, rerr := loadClient()
					if rerr != nil {
						return fmt.Errorf("auth dead: %w", rerr)
					}
					client = nc
				}
				return nil
			}
			seen := map[string]bool{}
			var mris []string
			add := func(mri string) {
				if mri == "" || mri == client.SelfMRI || seen[mri] {
					return
				}
				if !strings.HasPrefix(mri, "8:orgid:") && !strings.HasPrefix(mri, "28:") {
					return
				}
				seen[mri] = true
				mris = append(mris, mri)
			}
			for _, c := range chats {
				add(c.Creator)
				if i := strings.Index(c.ID, "@unq.gbl.spaces"); i > 0 {
					for part := range strings.SplitSeq(strings.TrimPrefix(c.ID[:i], "19:"), "_") {
						if len(part) == 36 {
							add("8:orgid:" + part)
						}
					}
				}
				for _, m := range c.Members {
					if m.FriendlyName == "" {
						add(m.MRI)
					}
				}
			}
			var names map[string]string
			if len(mris) > 0 {
				users, err := client.FetchShortProfile(ctx, mris)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: name resolution via middle tier failed: %v\n", err)
				} else {
					names = make(map[string]string, len(users))
					for _, u := range users {
						if u.DisplayName != "" {
							names[u.MRI] = u.DisplayName
						}
					}
				}
			}

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
				parseTime := func(m teams.Message) time.Time {
					raw := m.OriginalArrivalTime
					if raw == "" {
						raw = m.ComposeTime
					}
					t, _ := time.Parse(time.RFC3339Nano, raw)
					return t
				}
				sort.SliceStable(msgs, func(i, j int) bool {
					return parseTime(msgs[i]).Before(parseTime(msgs[j]))
				})
				idx := -1
				for i, m := range msgs {
					if m.ID == prev.LastMessageID {
						idx = i
						break
					}
				}

				threadName := func() string {
					if c.Title != nil && *c.Title != "" {
						return *c.Title
					}
					if c.MeetingInformation != nil && c.MeetingInformation.Subject != "" {
						return c.MeetingInformation.Subject
					}
					var memberNames []string
					for _, m := range c.Members {
						if m.MRI == client.SelfMRI {
							continue
						}
						name := m.FriendlyName
						if name == "" {
							name = names[m.MRI]
						}
						if name == "" {
							continue
						}
						memberNames = append(memberNames, name)
					}
					isGrp := c.ChatType == "chat" && !c.IsOneOnOne
					if len(memberNames) > 0 {
						out := memberNames
						if isGrp {
							out = make([]string, len(memberNames))
							for i, n := range memberNames {
								n = strings.TrimSpace(n)
								if j := strings.IndexAny(n, " \t"); j > 0 {
									n = n[:j]
								}
								out[i] = n
							}
						}
						if len(out) <= 3 {
							return strings.Join(out, ", ")
						}
						return fmt.Sprintf("%s, %s, %s +%d", out[0], out[1], out[2], len(out)-3)
					}
					if c.Creator != "" && c.Creator != client.SelfMRI {
						if n := names[c.Creator]; n != "" {
							return n
						}
					}
					if i := strings.Index(c.ID, "@unq.gbl.spaces"); i > 0 {
						for part := range strings.SplitSeq(strings.TrimPrefix(c.ID[:i], "19:"), "_") {
							if len(part) == 36 {
								mri := "8:orgid:" + part
								if mri != client.SelfMRI {
									if n := names[mri]; n != "" {
										return n
									}
								}
							}
						}
					}
					if c.LastMessage != nil && !c.IsLastMessageFromMe && c.LastMessage.IMDisplayName != "" {
						return c.LastMessage.IMDisplayName
					}
					switch {
					case c.ChatType == "meeting":
						return "Meeting chat"
					case c.IsOneOnOne:
						return "Direct chat"
					case c.ChatType == "chat":
						if n := len(c.Members); n > 0 {
							return fmt.Sprintf("Group chat (%d people)", n)
						}
						return "Group chat"
					}
					return "Conversation"
				}()
				isGroup := c.ChatType == "chat" && !c.IsOneOnOne

				fire := func(sender, content, messagetype, suffix string) {
					if sender == "" {
						sender = "Someone"
					}
					body := htmlToPlainText(content, messagetype, nil)
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
			return nil
		}

		if err := tick(); err != nil {
			return err
		}
		seeded = true
		t := time.NewTicker(watchInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				if err := tick(); err != nil {
					return err
				}
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

type jsonNotifier struct{ enc *json.Encoder }

func (j jsonNotifier) notify(title, subtitle, body string) error {
	return j.enc.Encode(struct {
		Time     string `json:"time"`
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
		Body     string `json:"body"`
	}{time.Now().UTC().Format(time.RFC3339), title, subtitle, body})
}
