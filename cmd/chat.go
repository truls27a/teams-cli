package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"

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
		ctx := context.Background()
		chats, err := client.ListChats(ctx)
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

		if err := saveChatIndex(chats); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save chat index: %v\n", err)
		}

		display := func(c teams.Chat) string {
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
			isGroup := c.ChatType == "chat" && !c.IsOneOnOne
			if len(memberNames) > 0 {
				out := memberNames
				if isGroup {
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
		}

		if jsonOutput {
			type item struct {
				ID             int    `json:"id"`
				Name           string `json:"name"`
				ConversationID string `json:"conversation_id"`
			}
			out := make([]item, 0, len(chats))
			for i, c := range chats {
				out = append(out, item{i + 1, display(c), c.ID})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		idW := max(len(strconv.Itoa(len(chats))), 2)
		for i := len(chats) - 1; i >= 0; i-- {
			c := chats[i]
			line := fmt.Sprintf("%-*d  %s", idW, i+1, display(c))
			if !c.IsRead && !c.IsLastMessageFromMe && !c.IsEmptyConversation {
				line += " *"
			}
			fmt.Println(line)
		}
		return nil
	},
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
		convID := args[0]
		if n, err := strconv.Atoi(convID); err == nil {
			idx, err := loadChatIndex()
			if err != nil {
				return fmt.Errorf("no chat index found — run `teams chat list` first")
			}
			if n < 1 || n > len(idx.Chats) {
				return fmt.Errorf("chat id %d out of range (1..%d) — run `teams chat list` to refresh", n, len(idx.Chats))
			}
			convID = idx.Chats[n-1]
		}
		msgs, _, err := client.ListMessages(context.Background(), convID, min(viewLimit, 200))
		if err != nil {
			return err
		}
		if len(msgs) > viewLimit {
			msgs = msgs[:viewLimit]
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

		type row struct {
			name, when, body, flag string
			deleted, edited        bool
			whenISO                string
		}
		rows := make([]row, 0, len(msgs))
		mriToName := map[string]string{}
		for _, m := range msgs {
			if m.From == "" || m.IMDisplayName == "" {
				continue
			}
			mri := m.From
			if i := strings.LastIndex(mri, "/"); i >= 0 {
				mri = mri[i+1:]
			}
			mriToName[mri] = m.IMDisplayName
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
			body := htmlToPlainText(m.Content, m.Messagetype, mriToName)
			if rawFiles, ok := m.Properties["files"].(string); ok && rawFiles != "" && rawFiles != "[]" {
				var files []struct {
					FileName string `json:"fileName"`
					FileType string `json:"fileType"`
					Title    string `json:"title"`
				}
				if json.Unmarshal([]byte(rawFiles), &files) == nil {
					var parts []string
					for _, f := range files {
						fname := f.FileName
						if fname == "" {
							fname = f.Title
						}
						ext := strings.ToLower(strings.TrimPrefix(f.FileType, "."))
						if ext == "" {
							if i := strings.LastIndex(fname, "."); i >= 0 {
								ext = strings.ToLower(fname[i+1:])
							}
						}
						kind := "file"
						switch ext {
						case "mov", "mp4", "webm", "avi", "mkv", "m4v":
							kind = "video"
						case "mp3", "wav", "m4a", "ogg", "flac", "aac":
							kind = "audio"
						case "jpg", "jpeg", "png", "gif", "bmp", "webp", "heic":
							kind = "image"
						}
						if fname != "" {
							parts = append(parts, fmt.Sprintf("[%s: %s]", kind, fname))
						} else {
							parts = append(parts, "["+kind+"]")
						}
					}
					if attach := strings.Join(parts, " "); attach != "" {
						if body == "" {
							body = attach
						} else {
							body = body + "\n" + attach
						}
					}
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
		var ws struct{ Row, Col, X, Y uint16 }
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
		width := 80
		if errno == 0 && ws.Col != 0 {
			width = int(ws.Col)
		}
		bodyW := max(width-2, 20)
		for i, r := range rows {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("%s, %s%s\n", r.name, r.when, r.flag)
			for ln := range strings.SplitSeq(r.body, "\n") {
				words := strings.Fields(ln)
				if len(words) == 0 {
					fmt.Println("  ")
					continue
				}
				cur, curW := "", 0
				flush := func() {
					fmt.Println("  " + cur)
					cur, curW = "", 0
				}
				for _, w := range words {
					ww := utf8.RuneCountInString(w)
					if curW == 0 {
						cur, curW = w, ww
						continue
					}
					if curW+1+ww > bodyW {
						flush()
						cur, curW = w, ww
						continue
					}
					cur += " " + w
					curW += 1 + ww
				}
				if cur != "" {
					flush()
				}
			}
		}

		for i := len(msgs) - 1; i >= 0; i-- {
			m := msgs[i]
			if m.ClientMessageID == "" {
				continue
			}
			if err := client.SetConsumptionHorizon(context.Background(), convID, m.ID, m.ClientMessageID); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to mark chat read: %v\n", err)
			}
			break
		}
		return nil
	},
}

var chatSendCmd = &cobra.Command{
	Use:   "send <conversation-id> [message]",
	Short: "Send a message",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := loadClient()
		if err != nil {
			return err
		}
		convID := args[0]
		if n, err := strconv.Atoi(convID); err == nil {
			idx, err := loadChatIndex()
			if err != nil {
				return fmt.Errorf("no chat index found — run `teams chat list` first")
			}
			if n < 1 || n > len(idx.Chats) {
				return fmt.Errorf("chat id %d out of range (1..%d) — run `teams chat list` to refresh", n, len(idx.Chats))
			}
			convID = idx.Chats[n-1]
		}

		var body string
		if len(args) == 2 {
			body = args[1]
		} else {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				body = string(b)
			} else {
				editor := os.Getenv("VISUAL")
				if editor == "" {
					editor = os.Getenv("EDITOR")
				}
				if editor == "" {
					editor = "vi"
				}
				f, err := os.CreateTemp("", "teams-message-*.md")
				if err != nil {
					return err
				}
				path := f.Name()
				f.Close()
				defer os.Remove(path)
				parts := strings.Fields(editor)
				ec := exec.Command(parts[0], append(parts[1:], path)...)
				ec.Stdin, ec.Stdout, ec.Stderr = os.Stdin, os.Stdout, os.Stderr
				if err := ec.Run(); err != nil {
					return err
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				body = string(b)
			}
		}
		body = strings.TrimSpace(body)
		if body == "" {
			return errors.New("aborting: empty message")
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
			Content:         body,
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

func htmlToPlainText(content, messagetype string, names map[string]string) string {
	label := func() string {
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

	if messagetype != "RichText/Html" {
		if l := label(); l != "" && strings.TrimSpace(content) == "" {
			return l
		}
		return strings.TrimSpace(content)
	}

	tag := regexp.MustCompile(`<[^>]+>`)
	attrAlt := regexp.MustCompile(`(?is)\balt="([^"]*)"`)

	blockquote := regexp.MustCompile(`(?is)<blockquote\b[^>]*>(.*?)</blockquote>\s*`)
	s := blockquote.ReplaceAllStringFunc(content, func(m string) string {
		inner := blockquote.FindStringSubmatch(m)[1]
		author := ""
		if mm := regexp.MustCompile(`(?is)<(?:strong|span|b)\b[^>]*\bitemprop="mri"[^>]*\bitemid="([^"]+)"`).FindStringSubmatch(inner); mm != nil {
			if n := names[mm[1]]; n != "" {
				author = n
			}
		}
		if author == "" {
			if mm := regexp.MustCompile(`(?is)<(?:strong|span|b)\b[^>]*\bitemprop="mri"[^>]*>(.*?)</(?:strong|span|b)>`).FindStringSubmatch(inner); mm != nil {
				author = strings.TrimSpace(tag.ReplaceAllString(mm[1], ""))
			}
		}
		preview := ""
		if mm := regexp.MustCompile(`(?is)<p\b[^>]*\bitemprop="preview"[^>]*>(.*?)</p>`).FindStringSubmatch(inner); mm != nil {
			preview = strings.TrimSpace(tag.ReplaceAllString(mm[1], ""))
		}
		if preview == "" {
			preview = strings.TrimSpace(tag.ReplaceAllString(inner, ""))
			preview = strings.Join(strings.Fields(preview), " ")
		}
		if preview == "" {
			return ""
		}
		if author != "" && author != "Display Name" {
			return "> " + author + ": " + preview + "\n"
		}
		return "> " + preview + "\n"
	})

	anchor := regexp.MustCompile(`(?is)<a\b[^>]*\bhref="([^"]+)"[^>]*>(.*?)</a>`)
	s = anchor.ReplaceAllStringFunc(s, func(m string) string {
		sub := anchor.FindStringSubmatch(m)
		href, text := sub[1], tag.ReplaceAllString(sub[2], "")
		text = strings.TrimSpace(text)
		if text == "" || text == href || strings.HasSuffix(text, "…") {
			return href
		}
		return text + " (" + href + ")"
	})

	uriObject := regexp.MustCompile(`(?is)<URIObject\b[^>]*\btype="([^"]+)"[^>]*>.*?</URIObject>`)
	s = uriObject.ReplaceAllStringFunc(s, func(m string) string {
		t := strings.ToLower(uriObject.FindStringSubmatch(m)[1])
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
	})

	s = regexp.MustCompile(`(?is)<video\b[^>]*>.*?</video>|<video\b[^>]*/?>`).ReplaceAllString(s, "[video]")
	s = regexp.MustCompile(`(?is)<audio\b[^>]*>.*?</audio>|<audio\b[^>]*/?>`).ReplaceAllString(s, "[audio]")
	s = regexp.MustCompile(`(?is)<emoji\b[^>]*>.*?</emoji>`).ReplaceAllStringFunc(s, func(m string) string {
		if a := attrAlt.FindStringSubmatch(m); a != nil && a[1] != "" {
			return a[1]
		}
		return ""
	})
	s = regexp.MustCompile(`(?is)<img\b[^>]*>`).ReplaceAllStringFunc(s, func(m string) string {
		alt := ""
		if a := attrAlt.FindStringSubmatch(m); a != nil {
			alt = strings.TrimSpace(a[1])
		}
		if it := regexp.MustCompile(`(?is)\bitemtype="([^"]*)"`).FindStringSubmatch(m); it != nil && strings.Contains(strings.ToLower(it[1]), "emoji") {
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
	s = regexp.MustCompile(`(?is)<attachment\b[^>]*>.*?</attachment>`).ReplaceAllString(s, "[attachment]")

	li := regexp.MustCompile(`(?is)<li\b[^>]*>(.*?)</li>`)
	listReplace := func(re *regexp.Regexp, ordered bool) {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			items := li.FindAllStringSubmatch(re.FindStringSubmatch(m)[1], -1)
			if len(items) == 0 {
				return ""
			}
			var b strings.Builder
			b.WriteString("\n")
			for i, mm := range items {
				if ordered {
					fmt.Fprintf(&b, "%d. ", i+1)
				} else {
					b.WriteString("- ")
				}
				b.WriteString(strings.TrimSpace(mm[1]))
				b.WriteString("\n")
			}
			return b.String()
		})
	}
	listReplace(regexp.MustCompile(`(?is)<ol\b[^>]*>(.*?)</ol>`), true)
	listReplace(regexp.MustCompile(`(?is)<ul\b[^>]*>(.*?)</ul>`), false)

	heading := regexp.MustCompile(`(?is)<h[1-6]\b[^>]*>(.*?)</h[1-6]>`)
	s = heading.ReplaceAllStringFunc(s, func(m string) string {
		txt := strings.TrimSpace(heading.FindStringSubmatch(m)[1])
		return "\n\n**" + txt + "**\n\n"
	})
	s = regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`).ReplaceAllString(s, "$1\n\n")
	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, "\n")

	bold := regexp.MustCompile(`(?is)<(b|strong)\b[^>]*>(.*?)</(?:b|strong)>`)
	s = bold.ReplaceAllStringFunc(s, func(m string) string {
		return "**" + bold.FindStringSubmatch(m)[2] + "**"
	})
	italic := regexp.MustCompile(`(?is)<(i|em)\b[^>]*>(.*?)</(?:i|em)>`)
	s = italic.ReplaceAllStringFunc(s, func(m string) string {
		return "*" + italic.FindStringSubmatch(m)[2] + "*"
	})
	s = tag.ReplaceAllString(s, "")
	s = strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
	).Replace(s)
	s = strings.ReplaceAll(s, "\r", "")
	s = regexp.MustCompile(`(?:[ \t]*\n){2,}`).ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)
	if s == "" {
		if l := label(); l != "" {
			return l
		}
	}
	return s
}

type chatIndex struct {
	Chats []string `json:"chats"`
}

func chatIndexPath() string {
	d, err := os.UserCacheDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".cache")
	}
	return filepath.Join(d, "teams-cli", "chat-index.json")
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

