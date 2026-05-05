package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"teams-cli/teams"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Manage chats",
}

var listAll bool

var chatListCmd = &cobra.Command{
	Use:   "list",
	Short: "List chats",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}
		chats, err := client.ListChats(context.Background())
		if err != nil {
			return err
		}

		if !listAll {
			filtered := chats[:0]
			for _, c := range chats {
				if strings.HasPrefix(c.ID, "19:meeting_") {
					continue
				}
				filtered = append(filtered, c)
			}
			chats = filtered
		}

		if jsonOutput {
			type item struct {
				Name string `json:"name"`
				Type string `json:"type"`
				ID   string `json:"id"`
			}
			out := make([]item, 0, len(chats))
			for _, c := range chats {
				out = append(out, item{chatDisplay(c, client.SelfMRI), chatType(c), c.ID})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		fmt.Printf("%-30s  %-8s  %s\n", "NAME", "TYPE", "ID")
		fmt.Println(strings.Repeat("-", 80))
		for _, c := range chats {
			fmt.Printf("%-30s  %-8s  %s\n", truncate(chatDisplay(c, client.SelfMRI), 30), chatType(c), c.ID)
		}
		return nil
	},
}

var (
	viewLimit int
	viewAll   bool
)

var chatViewCmd = &cobra.Command{
	Use:   "view <conversation-id>",
	Short: "View messages in a conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}
		var msgs []teams.Message
		if viewAll {
			msgs, err = fetchAllMessages(context.Background(), client, args[0])
		} else {
			pageSize := viewLimit
			if pageSize > 200 {
				pageSize = 200
			}
			msgs, _, err = client.ListMessages(context.Background(), args[0], pageSize)
			if err == nil && len(msgs) > viewLimit {
				msgs = msgs[:viewLimit]
			}
		}
		if err != nil {
			return err
		}

		// API returns newest-first; reverse for display
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}

		type row struct {
			name, when, body, flag string
			deleted, edited        bool
			whenISO               string
		}
		today := time.Now().Format("2006-01-02")
		rows := make([]row, 0, len(msgs))
		nameWidth := 0
		for _, m := range msgs {
			if strings.HasPrefix(m.Messagetype, "Control/") || strings.HasPrefix(m.Messagetype, "ThreadActivity/") {
				continue
			}
			raw := m.OriginalArrivalTime
			if raw == "" {
				raw = m.ComposeTime
			}
			when := raw
			if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
				if t.Format("2006-01-02") == today {
					when = t.Format("15:04")
				} else {
					when = t.Format("01-02 15:04")
				}
			}
			flag := ""
			_, deleted := m.Properties["deletetime"]
			_, edited := m.Properties["edittime"]
			if deleted {
				flag = " [deleted]"
			} else if edited {
				flag = " [edited]"
			}
			name := m.IMDisplayName
			if name == "" {
				name = m.From
				if i := strings.LastIndex(name, "/"); i >= 0 {
					name = name[i+1:]
				}
				if i := strings.LastIndex(name, ":"); i >= 0 {
					name = name[i+1:]
				}
				if name == "" {
					name = "(unknown)"
				} else if len(name) > 8 {
					name = name[:8]
				}
			}
			if n := len([]rune(name)); n > nameWidth {
				nameWidth = n
			}
			rows = append(rows, row{name, when, renderContent(m.Content, m.Messagetype), flag, deleted, edited, raw})
		}

		if jsonOutput {
			type item struct {
				Name    string `json:"name"`
				Time    string `json:"time"`
				Body    string `json:"body"`
				Deleted bool   `json:"deleted,omitempty"`
				Edited  bool   `json:"edited,omitempty"`
			}
			out := make([]item, 0, len(rows))
			for _, r := range rows {
				out = append(out, item{r.name, r.whenISO, r.body, r.deleted, r.edited})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		if nameWidth > 24 {
			nameWidth = 24
		}
		for _, r := range rows {
			lines := strings.Split(r.body, "\n")
			fmt.Printf("%-*s  %-11s  %s%s\n", nameWidth, truncate(r.name, nameWidth), r.when, lines[0], r.flag)
			for _, line := range lines[1:] {
				fmt.Printf("%-*s  %-11s  %s\n", nameWidth, "", "", line)
			}
		}
		return nil
	},
}

var chatSendCmd = &cobra.Command{
	Use:   "send <conversation-id> <message>",
	Short: "Send a message",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}
		clientMsgID := fmt.Sprintf("%d%03d", time.Now().UnixMilli(), rand.IntN(1000))
		resp, err := client.SendMessage(context.Background(), args[0], teams.SendMessageRequest{
			Content:         args[1],
			Messagetype:     "RichText/Html",
			Contenttype:     "Text",
			ClientMessageID: clientMsgID,
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				Sent bool  `json:"sent"`
				ID   int64 `json:"id"`
			}{true, resp.OriginalArrivalTime})
		}
		fmt.Printf("Sent (id: %d).\n", resp.OriginalArrivalTime)
		return nil
	},
}

func init() {
	chatListCmd.Flags().BoolVar(&listAll, "all", false, "include meeting threads and system feeds")
	chatViewCmd.Flags().IntVarP(&viewLimit, "limit", "n", 20, "number of messages to show")
	chatViewCmd.Flags().BoolVar(&viewAll, "all", false, "fetch all messages")
	chatCmd.AddCommand(chatListCmd, chatViewCmd, chatSendCmd)
}

func fetchAllMessages(ctx context.Context, client *teams.Client, convID string) ([]teams.Message, error) {
	msgs, syncState, err := client.ListMessages(ctx, convID, 200)
	if err != nil {
		return nil, err
	}
	for syncState != "" {
		page, next, err := client.ListMessagesPage(ctx, syncState)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, page...)
		syncState = next
	}
	return msgs, nil
}

var (
	tagRE    = regexp.MustCompile(`<[^>]+>`)
	anchorRE = regexp.MustCompile(`(?is)<a\b[^>]*\bhref="([^"]+)"[^>]*>(.*?)</a>`)
)

func renderContent(content, messagetype string) string {
	if messagetype == "RichText/Html" {
		s := anchorRE.ReplaceAllStringFunc(content, func(m string) string {
			sub := anchorRE.FindStringSubmatch(m)
			href, text := sub[1], tagRE.ReplaceAllString(sub[2], "")
			text = strings.TrimSpace(text)
			if text == "" || text == href || strings.HasSuffix(text, "…") {
				return href
			}
			return text + " (" + href + ")"
		})
		s = tagRE.ReplaceAllString(s, "")
		s = strings.NewReplacer(
			"&amp;", "&", "&lt;", "<", "&gt;", ">",
			"&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
		).Replace(s)
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(content)
}

func chatType(c teams.Chat) string {
	switch {
	case c.ChatType == "meeting":
		return "meeting"
	case c.IsOneOnOne:
		return "dm"
	case c.ChatType == "chat":
		return "group"
	}
	return "other"
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func chatDisplay(c teams.Chat, selfMRI string) string {
	if c.Title != nil && *c.Title != "" {
		return *c.Title
	}
	if c.MeetingInformation != nil && c.MeetingInformation.Subject != "" {
		return c.MeetingInformation.Subject
	}
	var names []string
	for _, m := range c.Members {
		if m.MRI == selfMRI || m.FriendlyName == "" {
			continue
		}
		names = append(names, m.FriendlyName)
	}
	switch {
	case len(names) == 0:
	case len(names) <= 3:
		return strings.Join(names, ", ")
	default:
		return fmt.Sprintf("%s, %s, %s +%d", names[0], names[1], names[2], len(names)-3)
	}
	if c.LastMessage != nil && !c.IsLastMessageFromMe && c.LastMessage.IMDisplayName != "" {
		return c.LastMessage.IMDisplayName
	}
	switch chatType(c) {
	case "dm":
		return "Direct chat"
	case "meeting":
		return "Meeting chat"
	case "group":
		if n := len(c.Members); n > 0 {
			return fmt.Sprintf("Group chat (%d people)", n)
		}
		return "Group chat"
	}
	return "Conversation"
}
