package teams

import (
	"context"
	"fmt"
	"net/url"
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
	if err := c.doMessaging(ctx, "GET", path, nil, &resp); err != nil {
		return nil, "", err
	}
	return resp.Messages, resp.Metadata.SyncState, nil
}

func (c *Client) ListMessagesPage(ctx context.Context, syncStateURL string) ([]Message, string, error) {
	var resp listMessagesResponse
	if err := c.doMessaging(ctx, "GET", syncStateURL, nil, &resp); err != nil {
		return nil, "", err
	}
	return resp.Messages, resp.Metadata.SyncState, nil
}

func (c *Client) SendMessage(ctx context.Context, conversationID string, req SendMessageRequest) (*SendMessageResponse, error) {
	path := "/v1/users/ME/conversations/" + url.PathEscape(conversationID) + "/messages"
	var resp SendMessageResponse
	if err := c.doMessaging(ctx, "POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
