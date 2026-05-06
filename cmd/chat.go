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

		names := resolveMissingNames(context.Background(), client, chats)

		if jsonOutput {
			type item struct {
				Name string `json:"name"`
				Type string `json:"type"`
				ID   string `json:"id"`
			}
			out := make([]item, 0, len(chats))
			for _, c := range chats {
				out = append(out, item{chatDisplay(c, client.SelfMRI, names), chatType(c), c.ID})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		fmt.Printf("%-30s  %-8s  %s\n", "NAME", "TYPE", "ID")
		fmt.Println(strings.Repeat("-", 80))
		for _, c := range chats {
			fmt.Printf("%-30s  %-8s  %s\n", truncate(chatDisplay(c, client.SelfMRI, names), 30), chatType(c), c.ID)
		}
		return nil
	},
}

func resolveMissingNames(ctx context.Context, client *teams.Client, chats []teams.Chat) map[string]string {
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
		for _, m := range c.Members {
			if m.FriendlyName != "" {
				continue
			}
			add(m.MRI)
		}
		for _, mri := range peerCandidates(c, client.SelfMRI) {
			add(mri)
		}
	}
	if len(mris) == 0 {
		return nil
	}
	users, err := client.FetchShortProfile(ctx, mris)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: name resolution via middle tier failed: %v\n", err)
		return nil
	}
	out := make(map[string]string, len(users))
	for _, u := range users {
		if u.DisplayName != "" {
			out[u.MRI] = u.DisplayName
		}
	}
	return out
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
		mriToName := map[string]string{}
		extractMRI := func(from string) string {
			if i := strings.LastIndex(from, "/"); i >= 0 {
				return from[i+1:]
			}
			return from
		}
		for _, m := range msgs {
			if m.From != "" && m.IMDisplayName != "" {
				mriToName[extractMRI(m.From)] = m.IMDisplayName
			}
		}
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
			rows = append(rows, row{name, when, renderContent(m.Content, m.Messagetype, mriToName), flag, deleted, edited, raw})
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
	tagRE        = regexp.MustCompile(`<[^>]+>`)
	anchorRE     = regexp.MustCompile(`(?is)<a\b[^>]*\bhref="([^"]+)"[^>]*>(.*?)</a>`)
	blockquoteRE = regexp.MustCompile(`(?is)<blockquote\b[^>]*>(.*?)</blockquote>\s*`)
	quoteAuthorRE = regexp.MustCompile(`(?is)<(?:strong|span|b)\b[^>]*\bitemprop="mri"[^>]*>(.*?)</(?:strong|span|b)>`)
	quoteAuthorMRIRE = regexp.MustCompile(`(?is)<(?:strong|span|b)\b[^>]*\bitemprop="mri"[^>]*\bitemid="([^"]+)"`)
	quotePreviewRE = regexp.MustCompile(`(?is)<p\b[^>]*\bitemprop="preview"[^>]*>(.*?)</p>`)
	imgRE          = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	videoRE        = regexp.MustCompile(`(?is)<video\b[^>]*>.*?</video>|<video\b[^>]*/?>`)
	audioRE        = regexp.MustCompile(`(?is)<audio\b[^>]*>.*?</audio>|<audio\b[^>]*/?>`)
	emojiRE        = regexp.MustCompile(`(?is)<emoji\b[^>]*>.*?</emoji>`)
	uriObjectRE    = regexp.MustCompile(`(?is)<URIObject\b[^>]*\btype="([^"]+)"[^>]*>.*?</URIObject>`)
	attachmentRE   = regexp.MustCompile(`(?is)<attachment\b[^>]*>.*?</attachment>`)
	attrAltRE      = regexp.MustCompile(`(?is)\balt="([^"]*)"`)
	attrItemtypeRE = regexp.MustCompile(`(?is)\bitemtype="([^"]*)"`)
)

func mediaLabelFromType(t string) string {
	t = strings.ToLower(t)
	switch {
	case strings.HasPrefix(t, "picture"), strings.HasPrefix(t, "image"):
		return "[image]"
	case strings.HasPrefix(t, "video"):
		return "[video]"
	case strings.HasPrefix(t, "audio"), strings.Contains(t, "voice"):
		return "[audio]"
	case strings.HasPrefix(t, "file"):
		return "[file]"
	}
	return "[attachment]"
}

func messagetypeLabel(messagetype string) string {
	mt := strings.ToLower(messagetype)
	switch {
	case strings.Contains(mt, "video"):
		return "[video]"
	case strings.Contains(mt, "audio"), strings.Contains(mt, "voice"):
		return "[audio]"
	case strings.Contains(mt, "image"), strings.Contains(mt, "picture"), strings.Contains(mt, "media_card"):
		return "[image]"
	case strings.Contains(mt, "file"), strings.Contains(mt, "generic"):
		return "[file]"
	case strings.Contains(mt, "media"):
		return "[attachment]"
	}
	return ""
}

func renderQuote(inner string, names map[string]string) string {
	author := ""
	if m := quoteAuthorMRIRE.FindStringSubmatch(inner); m != nil {
		if n := names[m[1]]; n != "" {
			author = n
		}
	}
	if author == "" {
		if m := quoteAuthorRE.FindStringSubmatch(inner); m != nil {
			author = strings.TrimSpace(tagRE.ReplaceAllString(m[1], ""))
		}
	}
	preview := ""
	if m := quotePreviewRE.FindStringSubmatch(inner); m != nil {
		preview = strings.TrimSpace(tagRE.ReplaceAllString(m[1], ""))
	}
	if preview == "" {
		preview = strings.TrimSpace(tagRE.ReplaceAllString(inner, ""))
		preview = strings.Join(strings.Fields(preview), " ")
	}
	if preview == "" {
		return ""
	}
	if author != "" && author != "Display Name" {
		return "> " + author + ": " + preview + "\n"
	}
	return "> " + preview + "\n"
}

func renderContent(content, messagetype string, names map[string]string) string {
	if messagetype == "RichText/Html" {
		s := blockquoteRE.ReplaceAllStringFunc(content, func(m string) string {
			sub := blockquoteRE.FindStringSubmatch(m)
			return renderQuote(sub[1], names)
		})
		s = anchorRE.ReplaceAllStringFunc(s, func(m string) string {
			sub := anchorRE.FindStringSubmatch(m)
			href, text := sub[1], tagRE.ReplaceAllString(sub[2], "")
			text = strings.TrimSpace(text)
			if text == "" || text == href || strings.HasSuffix(text, "…") {
				return href
			}
			return text + " (" + href + ")"
		})
		s = uriObjectRE.ReplaceAllStringFunc(s, func(m string) string {
			sub := uriObjectRE.FindStringSubmatch(m)
			return mediaLabelFromType(sub[1])
		})
		s = videoRE.ReplaceAllString(s, "[video]")
		s = audioRE.ReplaceAllString(s, "[audio]")
		s = emojiRE.ReplaceAllStringFunc(s, func(m string) string {
			if a := attrAltRE.FindStringSubmatch(m); a != nil && a[1] != "" {
				return a[1]
			}
			return ""
		})
		s = imgRE.ReplaceAllStringFunc(s, func(m string) string {
			alt := ""
			if a := attrAltRE.FindStringSubmatch(m); a != nil {
				alt = strings.TrimSpace(a[1])
			}
			if it := attrItemtypeRE.FindStringSubmatch(m); it != nil && strings.Contains(strings.ToLower(it[1]), "emoji") {
				if alt != "" {
					return alt
				}
				return ""
			}
			switch strings.ToLower(alt) {
			case "", "image", "picture", "photo":
				return "[image]"
			}
			return "[image: " + alt + "]"
		})
		s = attachmentRE.ReplaceAllString(s, "[attachment]")
		s = tagRE.ReplaceAllString(s, "")
		s = strings.NewReplacer(
			"&amp;", "&", "&lt;", "<", "&gt;", ">",
			"&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
		).Replace(s)
		for strings.Contains(s, "\n\n\n") {
			s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			if label := messagetypeLabel(messagetype); label != "" {
				return label
			}
		}
		return s
	}
	if label := messagetypeLabel(messagetype); label != "" && strings.TrimSpace(content) == "" {
		return label
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

func peerCandidates(c teams.Chat, selfMRI string) []string {
	var out []string
	if c.Creator != "" && c.Creator != selfMRI {
		out = append(out, c.Creator)
	}
	if i := strings.Index(c.ID, "@unq.gbl.spaces"); i > 0 {
		body := strings.TrimPrefix(c.ID[:i], "19:")
		for part := range strings.SplitSeq(body, "_") {
			if len(part) == 36 {
				mri := "8:orgid:" + part
				if mri != selfMRI {
					out = append(out, mri)
				}
			}
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func firstName(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i > 0 {
		return s[:i]
	}
	return s
}

func chatDisplay(c teams.Chat, selfMRI string, resolved map[string]string) string {
	if c.Title != nil && *c.Title != "" {
		return *c.Title
	}
	if c.MeetingInformation != nil && c.MeetingInformation.Subject != "" {
		return c.MeetingInformation.Subject
	}
	var names []string
	for _, m := range c.Members {
		if m.MRI == selfMRI {
			continue
		}
		name := m.FriendlyName
		if name == "" {
			name = resolved[m.MRI]
		}
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	if len(names) > 0 {
		isGroup := chatType(c) == "group"
		display := names
		if isGroup {
			display = make([]string, len(names))
			for i, n := range names {
				display[i] = firstName(n)
			}
		}
		if len(display) <= 3 {
			return strings.Join(display, ", ")
		}
		return fmt.Sprintf("%s, %s, %s +%d", display[0], display[1], display[2], len(display)-3)
	}
	for _, mri := range peerCandidates(c, selfMRI) {
		if name := resolved[mri]; name != "" {
			return name
		}
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
