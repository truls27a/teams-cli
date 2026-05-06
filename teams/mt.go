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

type User struct {
	MRI               string `json:"mri"`
	ObjectID          string `json:"objectId"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Email             string `json:"email"`
	UserPrincipalName string `json:"userPrincipalName"`
	UserType          string `json:"userType"`
	TenantName        string `json:"tenantName"`
	IsShortProfile    bool   `json:"isShortProfile"`
	Type              string `json:"type"`
}

type usersResponse struct {
	Type  string `json:"type"`
	Value []User `json:"value"`
}

func (c *Client) FetchShortProfile(ctx context.Context, mris []string) ([]User, error) {
	if len(mris) == 0 {
		return nil, nil
	}
	var resp usersResponse
	if err := c.doMT(ctx, "POST", "/users/fetchShortProfile", mris, &resp); err != nil {
		return nil, err
	}
	return resp.Value, nil
}

func (c *Client) doMT(ctx context.Context, method, path string, body any, out any) error {
	if c.MTBaseURL == "" {
		return fmt.Errorf("middle tier unavailable: no region known")
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := c.MTBaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.MTToken)
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
