package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Chat struct {
	ID                  string              `json:"id"`
	ChatType            string              `json:"chatType"`
	ThreadType          string              `json:"threadType"`
	ChatSubType         int                 `json:"chatSubType"`
	IsOneOnOne          bool                `json:"isOneOnOne"`
	IsLastMessageFromMe bool                `json:"isLastMessageFromMe"`
	IsRead              bool                `json:"isRead"`
	IsEmptyConversation bool                `json:"isEmptyConversation"`
	IsExternal          bool                `json:"isExternal"`
	IsMessagingDisabled bool                `json:"isMessagingDisabled"`
	Hidden              bool                `json:"hidden"`
	IsSticky            bool                `json:"isSticky"`
	Title               *string             `json:"title"`
	Creator             string              `json:"creator"`
	TenantID            string              `json:"tenantId"`
	CreatedAt           string              `json:"createdAt"`
	LastJoinAt          string              `json:"lastJoinAt"`
	Version             int64               `json:"version"`
	ThreadVersion       int64               `json:"threadVersion"`
	RosterVersion       int64               `json:"rosterVersion"`
	Members             []ChatMember        `json:"members"`
	LastMessage         *ChatLastMessage    `json:"lastMessage"`
	MeetingInformation  *MeetingInformation `json:"meetingInformation"`
	ConsumptionHorizon  *ConsumptionHorizon `json:"consumptionHorizon"`
	RelationshipState   *RelationshipState  `json:"relationshipState"`

	IsMigrated                bool           `json:"isMigrated"`
	IsGapDetectionEnabled     bool           `json:"isGapDetectionEnabled"`
	IsSmsThread               bool           `json:"isSmsThread"`
	IsLiveChatEnabled         bool           `json:"isLiveChatEnabled"`
	IsHighImportance          bool           `json:"isHighImportance"`
	InteropType               int            `json:"interopType"`
	InteropConversationStatus string         `json:"interopConversationStatus"`
	ImportState               string         `json:"importState"`
	TemplateType              string         `json:"templateType"`
	ProductContext            string         `json:"productContext"`
	MeetingPolicy             string         `json:"meetingPolicy"`
	IdentityMaskEnabled       bool           `json:"identityMaskEnabled"`
	HasTranscript             bool           `json:"hasTranscript"`
	IsConversationDeleted     bool           `json:"isConversationDeleted"`
	IsDisabled                bool           `json:"isDisabled"`
	ConversationBlockedAt     int64          `json:"conversationBlockedAt"`
	LastL2MessageIDNFS        int64          `json:"lastL2MessageIdNFS"`
	LastRcMetadataVersion     int64          `json:"lastRcMetadataVersion"`
	RetentionHorizon          *string        `json:"retentionHorizon"`
	RetentionHorizonV2        *string        `json:"retentionHorizonV2"`
	FileReferences            map[string]any `json:"fileReferences"`
}

type ChatMember struct {
	MRI              string `json:"mri"`
	ObjectID         string `json:"objectId"`
	TenantID         string `json:"tenantId,omitempty"`
	Role             string `json:"role"`
	FriendlyName     string `json:"friendlyName,omitempty"`
	IsMuted          bool   `json:"isMuted"`
	IsIdentityMasked bool   `json:"isIdentityMasked"`
	ShareHistoryTime string `json:"shareHistoryTime,omitempty"`
}

type MeetingInformation struct {
	Subject                   string         `json:"subject"`
	Location                  string         `json:"location"`
	StartTime                 string         `json:"startTime"`
	EndTime                   string         `json:"endTime"`
	ICalUID                   string         `json:"iCalUid"`
	IsCancelled               bool           `json:"isCancelled"`
	MeetingJoinURL            string         `json:"meetingJoinUrl"`
	OrganizerID               string         `json:"organizerId"`
	CoOrganizerIDs            []string       `json:"coOrganizerIds"`
	TenantID                  string         `json:"tenantId"`
	AppointmentType           int            `json:"appointmentType"`
	MeetingType               int            `json:"meetingType"`
	Scenario                  string         `json:"scenario"`
	IsCopyRestrictionEnforced bool           `json:"isCopyRestrictionEnforced"`
	GroupCopilotDetails       map[string]any `json:"groupCopilotDetails"`
	EnableMultiLingualMeeting bool           `json:"enableMultiLingualMeeting"`
	ExchangeID                *string        `json:"exchangeId"`
}

type ChatLastMessage struct {
	ID                      string  `json:"id"`
	Type                    string  `json:"type"`
	MessageType             string  `json:"messageType"`
	Content                 string  `json:"content"`
	ComposeTime             string  `json:"composeTime"`
	OriginalArrivalTime     string  `json:"originalArrivalTime"`
	ClientMessageID         string  `json:"clientMessageId"`
	ParentMessageID         string  `json:"parentMessageId"`
	ContainerID             string  `json:"containerId"`
	From                    string  `json:"from"`
	IMDisplayName           string  `json:"imDisplayName"`
	SequenceID              int64   `json:"sequenceId"`
	Version                 int64   `json:"version"`
	ThreadType              *string `json:"threadType"`
	FromDisplayNameInToken  *string `json:"fromDisplayNameInToken"`
	FromGivenNameInToken    *string `json:"fromGivenNameInToken"`
	FromFamilyNameInToken   *string `json:"fromFamilyNameInToken"`
	IsEscalationToNewPerson bool    `json:"isEscalationToNewPerson"`
}

type ConsumptionHorizon struct {
	OriginalArrivalTime int64  `json:"originalArrivalTime"`
	TimeStamp           int64  `json:"timeStamp"`
	ClientMessageID     string `json:"clientMessageId"`
}

type RelationshipState struct {
	InQuarantine     bool   `json:"inQuarantine"`
	HasImpersonation string `json:"hasImpersonation"`
}

type ChatsMetadata struct {
	SyncToken        string  `json:"syncToken"`
	ForwardSyncToken *string `json:"forwardSyncToken"`
	IsPartialData    bool    `json:"isPartialData"`
	HasMoreChats     bool    `json:"hasMoreChats"`
}

type chatsResponse struct {
	Chats        []Chat            `json:"chats"`
	Teams        []json.RawMessage `json:"teams"`
	Users        []json.RawMessage `json:"users"`
	PrivateFeeds []json.RawMessage `json:"privateFeeds"`
	Metadata     ChatsMetadata     `json:"metadata"`
}

func (c *Client) ListChats(ctx context.Context) ([]Chat, error) {
	const path = "/v1/teams/users/me?isPrefetch=false"
	var resp chatsResponse
	if err := c.doCSA(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Chats, nil
}

func (c *Client) doCSA(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := c.CSABaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.CSAToken)
	req.Header.Set("Accept", "application/json")
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
		return &APIError{method, url, resp.StatusCode, strings.TrimSpace(string(b))}
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
