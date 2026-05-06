package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Message struct {
	ID                  string            `json:"id"`
	SequenceID          int64             `json:"sequenceId"`
	Type                string            `json:"type"`
	Messagetype         string            `json:"messagetype"`
	Contenttype         string            `json:"contenttype"`
	Content             string            `json:"content"`
	From                string            `json:"from"`
	IMDisplayName       string            `json:"imdisplayname"`
	ClientMessageID     string            `json:"clientmessageid"`
	ConversationID      string            `json:"conversationid"`
	ConversationLink    string            `json:"conversationLink"`
	ComposeTime         string            `json:"composetime"`
	OriginalArrivalTime string            `json:"originalarrivaltime"`
	Version             string            `json:"version"`
	ParentMessageID     string            `json:"parentmessageid,omitempty"`
	SkypeEditedID       string            `json:"skypeeditedid,omitempty"`
	AMSReferences       []string          `json:"amsreferences"`
	Properties          map[string]any    `json:"properties"`
}

type listMessagesResponse struct {
	Messages []Message `json:"messages"`
	TenantID string    `json:"tenantId"`
	Metadata struct {
		TotalCount int    `json:"totalCount"`
		SyncState  string `json:"syncState"`
	} `json:"_metadata"`
}

type SendMessageRequest struct {
	Content         string            `json:"content"`
	Messagetype     string            `json:"messagetype"`
	Contenttype     string            `json:"contenttype"`
	ClientMessageID string            `json:"clientmessageid"`
	IMDisplayName   string            `json:"imdisplayname,omitempty"`
	ParentMessageID string            `json:"parentmessageid,omitempty"`
	Properties      map[string]string `json:"properties,omitempty"`
	AMSReferences   []string          `json:"amsreferences,omitempty"`
}

type SendMessageResponse struct {
	OriginalArrivalTime int64 `json:"OriginalArrivalTime"`
}

func (c *Client) ListMessages(ctx context.Context, conversationID string, pageSize int) ([]Message, string, error) {
	path := fmt.Sprintf("/v1/users/ME/conversations/%s/messages?pageSize=%d&view=msnp24Equivalent",
		url.PathEscape(conversationID), pageSize)
	var resp listMessagesResponse
	if err := c.doChatSvc(ctx, "GET", path, nil, &resp); err != nil {
		return nil, "", err
	}
	return resp.Messages, resp.Metadata.SyncState, nil
}

func (c *Client) SetConsumptionHorizon(ctx context.Context, conversationID, messageID, clientMessageID string) error {
	path := "/v1/users/ME/conversations/" + url.PathEscape(conversationID) + "/properties?name=consumptionhorizon"
	value := fmt.Sprintf("%s;%d;%s", messageID, time.Now().UnixMilli(), clientMessageID)
	body := map[string]string{"consumptionhorizon": value}
	return c.doChatSvc(ctx, "PUT", path, body, nil)
}

func (c *Client) SendMessage(ctx context.Context, conversationID string, req SendMessageRequest) (*SendMessageResponse, error) {
	path := "/v1/users/ME/conversations/" + url.PathEscape(conversationID) + "/messages"
	var resp SendMessageResponse
	if err := c.doChatSvc(ctx, "POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) doChatSvc(ctx context.Context, method, pathOrURL string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := pathOrURL
	if !strings.HasPrefix(pathOrURL, "http") {
		url = c.MessagingBaseURL + pathOrURL
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authentication", "skypetoken="+c.SkypeToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("BehaviorOverride", "redirectAs404")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %d %s", method, url, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
