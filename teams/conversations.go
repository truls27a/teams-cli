package teams

import (
	"context"
	"fmt"
	"net/url"
)

type ThreadProperties struct {
	ThreadType        string `json:"threadType"`
	ProductThreadType string `json:"productThreadType"`
	Topic             string `json:"topic"`
	Creator           string `json:"creator"`
	MemberCount       string `json:"memberCount"`
	LastSequenceID    string `json:"lastSequenceId"`
	Version           string `json:"version"`
	CreateAt          string `json:"createdat"`
	LastJoinAt        string `json:"lastjoinat"`
	OriginalThreadID  string `json:"originalThreadId"`
	TenantID          string `json:"tenantid"`
	RosterVersion     int64  `json:"rosterVersion"`
	IsCreator         bool   `json:"isCreator"`
	IsStickyThread    string `json:"isStickyThread"`
	GapDetection      string `json:"gapDetectionEnabled"`
	Hidden            string `json:"hidden"`
	Picture           string `json:"picture"`
}

type RelationshipState struct {
	InQuarantine    bool   `json:"inQuarantine"`
	HasImpersonation string `json:"hasImpersonation"`
}

type MemberProperties struct {
	Role                 string             `json:"role"`
	IsReader             bool               `json:"isReader"`
	IsIdentityMasked     bool               `json:"isIdentityMasked"`
	MemberExpirationTime int64              `json:"memberExpirationTime"`
	Interest             string             `json:"interest"`
	RelationshipState    *RelationshipState `json:"relationshipState"`
}

type Conversation struct {
	ID                        string            `json:"id"`
	Type                      string            `json:"type"`
	TargetLink                string            `json:"targetLink"`
	MessagesURL               string            `json:"messages"`
	Version                   int64             `json:"version"`
	LastUpdatedMessageID      int64             `json:"lastUpdatedMessageId"`
	LastUpdatedMessageVersion int64             `json:"lastUpdatedMessageVersion"`
	LastRcMetadataVersion     int64             `json:"lastRcMetadataVersion"`
	LastMessage               *Message          `json:"lastMessage"`
	Properties                map[string]any    `json:"properties"`
	ThreadProperties          *ThreadProperties `json:"threadProperties"`
	MemberProperties          *MemberProperties `json:"memberProperties"`
}

type listConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	Metadata      struct {
		TotalCount int    `json:"totalCount"`
		SyncState  string `json:"syncState"`
	} `json:"_metadata"`
}

func (c *Client) ListConversations(ctx context.Context, pageSize int) ([]Conversation, error) {
	path := fmt.Sprintf("/v1/users/ME/conversations?pageSize=%d&view=msnp24Equivalent", pageSize)
	var resp listConversationsResponse
	if err := c.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Conversations, nil
}

func (c *Client) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	path := "/v1/users/ME/conversations/" + url.PathEscape(id) + "?view=msnp24Equivalent"
	var conv Conversation
	if err := c.do(ctx, "GET", path, nil, &conv); err != nil {
		return nil, err
	}
	return &conv, nil
}
