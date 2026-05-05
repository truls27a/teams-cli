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
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(convs)
		}
		fmt.Printf("%-12s  %-10s  %s\n", "TYPE", "ID", "DISPLAY")
		fmt.Println(strings.Repeat("-", 72))
		for _, c := range convs {
			fmt.Printf("%-12s  %-48s  %s\n", convType(c), c.ID, convDisplay(c))
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

		for _, m := range msgs {
			if strings.HasPrefix(m.Messagetype, "Control/") {
				continue
			}
			when := m.OriginalArrivalTime
			if when == "" {
				when = m.ComposeTime
			}
			if t, err := time.Parse(time.RFC3339Nano, when); err == nil {
				when = t.Format("2006-01-02 15:04")
			}
			flag := ""
			if m.Properties["deletetime"] != "" {
				flag = " [deleted]"
			} else if m.Properties["edittime"] != "" {
				flag = " [edited]"
			}
			fmt.Printf("[%s] %s%s\n", when, m.IMDisplayName, flag)
			for _, line := range strings.Split(renderContent(m.Content, m.Messagetype), "\n") {
				fmt.Printf("    %s\n", line)
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

var tagRE = regexp.MustCompile(`<[^>]+>`)

func renderContent(content, messagetype string) string {
	if messagetype == "RichText/Html" {
		s := tagRE.ReplaceAllString(content, "")
		s = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'").Replace(s)
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(content)
}

func convType(c teams.Conversation) string {
	switch {
	case strings.HasPrefix(c.ID, "8:orgid:"):
		return "1:1"
	case strings.HasPrefix(c.ID, "8:live:"):
		return "1:1(ext)"
	case strings.HasPrefix(c.ID, "48:bot:"):
		return "Bot"
	case c.ThreadProperties != nil:
		switch c.ThreadProperties.ProductThreadType {
		case "OneToOneChat":
			return "1:1(fed)"
		case "Chat":
			return "Group"
		case "TopicThread":
			return "Channel"
		}
	}
	return "?"
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
