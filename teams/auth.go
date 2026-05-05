package teams

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ClientID = "1fec8e78-bce4-4aaf-ab1b-5451cc387264"
	Scope    = "https://api.spaces.skype.com/.default offline_access"
	tenant   = "organizations"
)

type DeviceCodeInfo struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type SkypeAuth struct {
	SkypeToken string `json:"skype_token"`
	ExpiresIn  int    `json:"expires_in"`
	BaseURL    string `json:"base_url"`
}

func RequestDeviceCode(ctx context.Context) (*DeviceCodeInfo, error) {
	resp, err := http.PostForm(
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/devicecode", tenant),
		url.Values{"client_id": {ClientID}, "scope": {Scope}},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info DeviceCodeInfo
	return &info, json.NewDecoder(resp.Body).Decode(&info)
}

func PollDeviceCode(ctx context.Context, info *DeviceCodeInfo) (*Tokens, error) {
	interval := info.Interval
	if interval == 0 {
		interval = 5
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}

		tr, err := http.PostForm(
			fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant),
			url.Values{
				"client_id":   {ClientID},
				"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
				"device_code": {info.DeviceCode},
			},
		)
		if err != nil {
			return nil, err
		}
		b, _ := io.ReadAll(tr.Body)
		tr.Body.Close()

		var tok struct {
			Tokens
			Error string `json:"error"`
		}
		if err := json.Unmarshal(b, &tok); err != nil {
			return nil, err
		}
		switch tok.Error {
		case "":
			return &tok.Tokens, nil
		case "authorization_pending":
		case "slow_down":
			interval += 5
		default:
			return nil, fmt.Errorf("token error: %s", tok.Error)
		}
	}
}

func ExchangeSkypeToken(accessToken string) (*SkypeAuth, error) {
	req, err := http.NewRequest("POST", "https://teams.microsoft.com/api/authsvc/v1.0/authz", strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Length", "0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var authz struct {
		Tokens struct {
			SkypeToken string `json:"skypeToken"`
			ExpiresIn  int    `json:"expiresIn"`
		} `json:"tokens"`
		RegionGtms struct {
			ChatService string `json:"chatService"`
		} `json:"regionGtms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authz); err != nil {
		return nil, err
	}
	return &SkypeAuth{
		SkypeToken: authz.Tokens.SkypeToken,
		ExpiresIn:  authz.Tokens.ExpiresIn,
		BaseURL:    authz.RegionGtms.ChatService,
	}, nil
}

func RefreshAccessToken(refreshToken string) (*Tokens, error) {
	resp, err := http.PostForm(
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant),
		url.Values{
			"client_id":     {ClientID},
			"grant_type":    {"refresh_token"},
			"refresh_token": {refreshToken},
			"scope":         {Scope},
		},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tok struct {
		Tokens
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.Error != "" {
		return nil, fmt.Errorf("refresh error: %s", tok.Error)
	}
	return &tok.Tokens, nil
}
