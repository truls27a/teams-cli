package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"teams/teams"

	"github.com/spf13/cobra"
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

		if err := saveChatIndex(chats); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save chat index: %v\n", err)
		}

		if jsonOutput {
			type item struct {
				ID             int    `json:"id"`
				Name           string `json:"name"`
				ConversationID string `json:"conversation_id"`
			}
			out := make([]item, 0, len(chats))
			for i, c := range chats {
				out = append(out, item{i + 1, chatDisplay(c, client.SelfMRI, names), c.ID})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		idW := len(strconv.Itoa(len(chats)))
		if idW < 2 {
			idW = 2
		}
		fmt.Printf("%-*s  %s\n", idW, "ID", "NAME")
		for i, c := range chats {
			fmt.Printf("%-*d  %s\n", idW, i+1, chatDisplay(c, client.SelfMRI, names))
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

var viewLimit int

var chatViewCmd = &cobra.Command{
	Use:   "view <conversation-id>",
	Short: "View messages in a conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}
		pageSize := viewLimit
		if pageSize > 200 {
			pageSize = 200
		}
		convID, err := resolveChatID(args[0])
		if err != nil {
			return err
		}
		msgs, _, err := client.ListMessages(context.Background(), convID, pageSize)
		if err != nil {
			return err
		}
		if len(msgs) > viewLimit {
			msgs = msgs[:viewLimit]
		}

		sort.SliceStable(msgs, func(i, j int) bool {
			return messageTime(msgs[i]).Before(messageTime(msgs[j]))
		})

		type row struct {
			name, when, body, flag string
			deleted, edited        bool
			whenISO                string
		}
		rows := make([]row, 0, len(msgs))
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
				when = t.Local().Format("2006-01-02 15:04")
			}
			flag := ""
			_, deleted := m.Properties["deletetime"]
			_, edited := m.Properties["edittime"]
			if deleted {
				flag = " (deleted)"
			} else if edited {
				flag = " (edited)"
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
			body := renderContent(m.Content, m.Messagetype, mriToName, !jsonOutput)
			if attach := renderAttachments(m.Properties); attach != "" {
				if body == "" {
					body = attach
				} else {
					body = body + "\n" + attach
				}
			}
			rows = append(rows, row{name, when, body, flag, deleted, edited, raw})
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
		for i, r := range rows {
			if i > 0 {
				fmt.Println("---")
			}
			fmt.Printf("%s, %s%s\n", r.name, r.when, r.flag)
			fmt.Println(r.body)
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
		convID, err := resolveChatID(args[0])
		if err != nil {
			return err
		}
		clientMsgID := fmt.Sprintf("%d%03d", time.Now().UnixMilli(), rand.IntN(1000))
		displayName := ""
		if users, err := client.FetchShortProfile(context.Background(), []string{client.SelfMRI}); err == nil {
			for _, u := range users {
				if u.MRI == client.SelfMRI {
					displayName = u.DisplayName
					break
				}
			}
		}
		_, err = client.SendMessage(context.Background(), convID, teams.SendMessageRequest{
			Content:         args[1],
			Messagetype:     "RichText/Html",
			Contenttype:     "Text",
			ClientMessageID: clientMsgID,
			IMDisplayName:   displayName,
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				Sent bool `json:"sent"`
			}{true})
		}
		fmt.Println("Sent.")
		return nil
	},
}

func init() {
	chatListCmd.Flags().BoolVar(&listAll, "all", false, "include meeting threads and system feeds")
	chatViewCmd.Flags().IntVarP(&viewLimit, "limit", "n", 20, "number of messages to show")
	chatCmd.AddCommand(chatListCmd, chatViewCmd, chatSendCmd)
}

var (
	tagRE            = regexp.MustCompile(`<[^>]+>`)
	anchorRE         = regexp.MustCompile(`(?is)<a\b[^>]*\bhref="([^"]+)"[^>]*>(.*?)</a>`)
	blockquoteRE     = regexp.MustCompile(`(?is)<blockquote\b[^>]*>(.*?)</blockquote>\s*`)
	quoteAuthorRE    = regexp.MustCompile(`(?is)<(?:strong|span|b)\b[^>]*\bitemprop="mri"[^>]*>(.*?)</(?:strong|span|b)>`)
	quoteAuthorMRIRE = regexp.MustCompile(`(?is)<(?:strong|span|b)\b[^>]*\bitemprop="mri"[^>]*\bitemid="([^"]+)"`)
	quotePreviewRE   = regexp.MustCompile(`(?is)<p\b[^>]*\bitemprop="preview"[^>]*>(.*?)</p>`)
	imgRE            = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	videoRE          = regexp.MustCompile(`(?is)<video\b[^>]*>.*?</video>|<video\b[^>]*/?>`)
	audioRE          = regexp.MustCompile(`(?is)<audio\b[^>]*>.*?</audio>|<audio\b[^>]*/?>`)
	emojiRE          = regexp.MustCompile(`(?is)<emoji\b[^>]*>.*?</emoji>`)
	uriObjectRE      = regexp.MustCompile(`(?is)<URIObject\b[^>]*\btype="([^"]+)"[^>]*>.*?</URIObject>`)
	attachmentRE     = regexp.MustCompile(`(?is)<attachment\b[^>]*>.*?</attachment>`)
	attrAltRE        = regexp.MustCompile(`(?is)\balt="([^"]*)"`)
	attrItemtypeRE   = regexp.MustCompile(`(?is)\bitemtype="([^"]*)"`)
	olRE             = regexp.MustCompile(`(?is)<ol\b[^>]*>(.*?)</ol>`)
	ulRE             = regexp.MustCompile(`(?is)<ul\b[^>]*>(.*?)</ul>`)
	liRE             = regexp.MustCompile(`(?is)<li\b[^>]*>(.*?)</li>`)
	pRE              = regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`)
	brRE             = regexp.MustCompile(`(?i)<br\s*/?>`)
	headingRE        = regexp.MustCompile(`(?is)<h[1-6]\b[^>]*>(.*?)</h[1-6]>`)
	boldRE           = regexp.MustCompile(`(?is)<(b|strong)\b[^>]*>(.*?)</(?:b|strong)>`)
	italicRE         = regexp.MustCompile(`(?is)<(i|em)\b[^>]*>(.*?)</(?:i|em)>`)
	blankLinesRE     = regexp.MustCompile(`(?:[ \t]*\n){2,}`)
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

func renderList(inner string, ordered bool) string {
	items := liRE.FindAllStringSubmatch(inner, -1)
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	for i, m := range items {
		if ordered {
			fmt.Fprintf(&b, "%d. ", i+1)
		} else {
			b.WriteString("- ")
		}
		b.WriteString(strings.TrimSpace(m[1]))
		b.WriteString("\n")
	}
	return b.String()
}

func renderContent(content, messagetype string, names map[string]string, ansi bool) string {
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
			case "", "image", "picture", "photo", "bild":
				return "[image]"
			}
			return "[image: " + alt + "]"
		})
		s = attachmentRE.ReplaceAllString(s, "[attachment]")

		s = olRE.ReplaceAllStringFunc(s, func(m string) string {
			return renderList(olRE.FindStringSubmatch(m)[1], true)
		})
		s = ulRE.ReplaceAllStringFunc(s, func(m string) string {
			return renderList(ulRE.FindStringSubmatch(m)[1], false)
		})
		s = headingRE.ReplaceAllStringFunc(s, func(m string) string {
			txt := strings.TrimSpace(headingRE.FindStringSubmatch(m)[1])
			if ansi {
				return "\n\n\x1b[1m" + txt + "\x1b[22m\n\n"
			}
			return "\n\n**" + txt + "**\n\n"
		})
		s = pRE.ReplaceAllString(s, "$1\n\n")
		s = brRE.ReplaceAllString(s, "\n")
		s = boldRE.ReplaceAllStringFunc(s, func(m string) string {
			txt := boldRE.FindStringSubmatch(m)[2]
			if ansi {
				return "\x1b[1m" + txt + "\x1b[22m"
			}
			return "**" + txt + "**"
		})
		s = italicRE.ReplaceAllStringFunc(s, func(m string) string {
			txt := italicRE.FindStringSubmatch(m)[2]
			if ansi {
				return "\x1b[3m" + txt + "\x1b[23m"
			}
			return "*" + txt + "*"
		})
		s = tagRE.ReplaceAllString(s, "")
		s = strings.NewReplacer(
			"&amp;", "&", "&lt;", "<", "&gt;", ">",
			"&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
		).Replace(s)
		s = strings.ReplaceAll(s, "\r", "")
		s = blankLinesRE.ReplaceAllString(s, "\n\n")
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

func messageTime(m teams.Message) time.Time {
	raw := m.OriginalArrivalTime
	if raw == "" {
		raw = m.ComposeTime
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t
	}
	return time.Time{}
}

func renderAttachments(props map[string]any) string {
	raw, ok := props["files"].(string)
	if !ok || raw == "" || raw == "[]" {
		return ""
	}
	var files []struct {
		FileName string `json:"fileName"`
		FileType string `json:"fileType"`
		Title    string `json:"title"`
	}
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		return ""
	}
	var parts []string
	for _, f := range files {
		name := f.FileName
		if name == "" {
			name = f.Title
		}
		kind := fileKind(f.FileType, name)
		if name != "" {
			parts = append(parts, fmt.Sprintf("[%s: %s]", kind, name))
		} else {
			parts = append(parts, "["+kind+"]")
		}
	}
	return strings.Join(parts, " ")
}

func fileKind(ext, name string) string {
	e := strings.ToLower(strings.TrimPrefix(ext, "."))
	if e == "" {
		if i := strings.LastIndex(name, "."); i >= 0 {
			e = strings.ToLower(name[i+1:])
		}
	}
	switch e {
	case "mov", "mp4", "webm", "avi", "mkv", "m4v":
		return "video"
	case "mp3", "wav", "m4a", "ogg", "flac", "aac":
		return "audio"
	case "jpg", "jpeg", "png", "gif", "bmp", "webp", "heic":
		return "image"
	}
	return "file"
}

func firstName(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i > 0 {
		return s[:i]
	}
	return s
}

func chatIndexPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "teams", "chat-index.json")
}

type chatIndex struct {
	Chats []string `json:"chats"`
}

func saveChatIndex(chats []teams.Chat) error {
	idx := chatIndex{Chats: make([]string, len(chats))}
	for i, c := range chats {
		idx.Chats[i] = c.ID
	}
	path := chatIndexPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(idx, "", "  ")
	return os.WriteFile(path, b, 0600)
}

func loadChatIndex() (*chatIndex, error) {
	b, err := os.ReadFile(chatIndexPath())
	if err != nil {
		return nil, err
	}
	var idx chatIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func resolveChatID(arg string) (string, error) {
	if n, err := strconv.Atoi(arg); err == nil {
		idx, err := loadChatIndex()
		if err != nil {
			return "", fmt.Errorf("no chat index found — run `teams chat list` first")
		}
		if n < 1 || n > len(idx.Chats) {
			return "", fmt.Errorf("chat id %d out of range (1..%d) — run `teams chat list` to refresh", n, len(idx.Chats))
		}
		return idx.Chats[n-1], nil
	}
	return arg, nil
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
