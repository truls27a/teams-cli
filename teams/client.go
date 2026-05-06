package teams

import "net/http"

type Client struct {
	MessagingBaseURL string
	SkypeToken       string

	CSABaseURL string
	CSAToken   string

	MTBaseURL string
	MTToken   string

	SelfMRI string

	HTTP *http.Client
}

func NewClient(messagingBaseURL, skypeToken, csaToken, region, mtToken, selfMRI string) *Client {
	mtBase := ""
	if region != "" {
		mtBase = "https://teams.microsoft.com/api/mt/" + region + "/beta"
	}
	return &Client{
		MessagingBaseURL: messagingBaseURL,
		SkypeToken:       skypeToken,
		CSABaseURL:       CSABaseURL,
		CSAToken:         csaToken,
		MTBaseURL:        mtBase,
		MTToken:          mtToken,
		SelfMRI:          selfMRI,
		HTTP:             &http.Client{},
	}
}
