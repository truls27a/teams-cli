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
	BaseURL    string
	SkypeToken string
	HTTP       *http.Client
}

func NewClient(baseURL, skypeToken string) *Client {
	return &Client{
		BaseURL:    baseURL,
		SkypeToken: skypeToken,
		HTTP:       &http.Client{},
	}
}

func (c *Client) do(ctx context.Context, method, pathOrURL string, body any, out any) error {
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
		url = c.BaseURL + pathOrURL
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
