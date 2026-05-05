package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

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

func (c *Client) doMessaging(ctx context.Context, method, pathOrURL string, body any, out any) error {
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

func (c *Client) doCSA(ctx context.Context, method, path string, body any, out any) error {
	return c.doBearer(ctx, method, c.CSABaseURL+path, c.CSAToken, body, out)
}

func (c *Client) doMT(ctx context.Context, method, path string, body any, out any) error {
	if c.MTBaseURL == "" {
		return fmt.Errorf("middle tier unavailable: no region known")
	}
	return c.doBearer(ctx, method, c.MTBaseURL+path, c.MTToken, body, out)
}

func (c *Client) doBearer(ctx context.Context, method, url, token string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
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
		return fmt.Errorf("%s %s: %d %s", method, url, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
