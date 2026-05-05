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
	Short: "List conversations",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}
		convs, err := client.ListConversations(context.Background(), 100)
		if err != nil {
			return err
		}

		if !listAll {
			filtered := convs[:0]
			for _, c := range convs {
				if strings.HasPrefix(c.ID, "48:") || strings.HasPrefix(c.ID, "19:meeting_") {
					continue
				}
				filtered = append(filtered, c)
			}
			convs = filtered
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(convs)
		}

		fmt.Printf("%-30s  %-8s  %s\n", "NAME", "TYPE", "ID")
		fmt.Println(strings.Repeat("-", 80))
		for _, c := range convs {
			fmt.Printf("%-30s  %-8s  %s\n", truncate(convDisplay(c), 30), convType(c), c.ID)
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

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(msgs)
		}

		type row struct{ name, when, body, flag string }
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
			if _, ok := m.Properties["deletetime"]; ok {
				flag = " [deleted]"
			} else if _, ok := m.Properties["edittime"]; ok {
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
			rows = append(rows, row{name, when, renderContent(m.Content, m.Messagetype), flag})
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
			return enc.Encode(resp)
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

func convType(c teams.Conversation) string {
	switch {
	case strings.HasPrefix(c.ID, "8:orgid:"), strings.HasPrefix(c.ID, "8:live:"):
		return "dm"
	case strings.HasPrefix(c.ID, "48:bot:"):
		return "bot"
	case c.ThreadProperties != nil:
		switch c.ThreadProperties.ProductThreadType {
		case "OneToOneChat":
			return "dm"
		case "Chat":
			return "group"
		case "TopicThread":
			return "channel"
		case "StreamOfNotifications":
			return "feed"
		}
	}
	return "other"
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func convDisplay(c teams.Conversation) string {
	if c.ThreadProperties != nil && c.ThreadProperties.Topic != "" {
		return c.ThreadProperties.Topic
	}
	if c.LastMessage != nil && c.LastMessage.IMDisplayName != "" {
		return c.LastMessage.IMDisplayName
	}
	return ""
}
