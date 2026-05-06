package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"teams/teams"
)

type storedAuth struct {
	RefreshToken string `json:"refresh_token"`

	SpacesToken  string    `json:"spaces_token"`
	SpacesExpiry time.Time `json:"spaces_expiry"`

	SkypeToken       string    `json:"skype_token"`
	SkypeExpiry      time.Time `json:"skype_expiry"`
	MessagingBaseURL string    `json:"messaging_base_url"`
	Region           string    `json:"region"`

	CSAToken  string    `json:"csa_token"`
	CSAExpiry time.Time `json:"csa_expiry"`

	SelfMRI string `json:"self_mri"`
}

func authPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "teams", "auth.json")
}

func loadAuth() (*storedAuth, error) {
	b, err := os.ReadFile(authPath())
	if err != nil {
		return nil, err
	}
	var a storedAuth
	return &a, json.Unmarshal(b, &a)
}

func saveAuth(a *storedAuth) error {
	path := authPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(a, "", "  ")
	return os.WriteFile(path, b, 0600)
}

func resolveSelfMRI(csaToken, skypeToken string) string {
	if claims, err := teams.DecodeJWTClaims(csaToken); err == nil {
		if oid, ok := claims["oid"].(string); ok && oid != "" {
			return "8:orgid:" + oid
		}
	}
	if claims, err := teams.DecodeJWTClaims(skypeToken); err == nil {
		if id, ok := claims["skypeid"].(string); ok && id != "" {
			return id
		}
	}
	return ""
}

func regionFromBaseURL(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Host
	if i := strings.Index(host, "."); i > 0 {
		return host[:i]
	}
	return ""
}

func loadClient() (*teams.Client, error) {
	a, err := loadAuth()
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("not logged in — run: teams auth login")
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dirty := false

	skypeStale := a.SkypeExpiry.Before(now.Add(60*time.Second)) ||
		a.MessagingBaseURL == "" || a.Region == ""
	if skypeStale {
		tok, err := teams.RefreshAccessToken(a.RefreshToken, teams.SkypeScope)
		if err != nil {
			return nil, fmt.Errorf("refresh spaces token: %w (run: teams auth login)", err)
		}
		skype, err := teams.ExchangeSkypeToken(tok.AccessToken)
		if err != nil {
			return nil, err
		}
		if tok.RefreshToken != "" {
			a.RefreshToken = tok.RefreshToken
		}
		a.SpacesToken = tok.AccessToken
		a.SpacesExpiry = now.Add(time.Duration(tok.ExpiresIn) * time.Second)
		a.SkypeToken = skype.SkypeToken
		a.SkypeExpiry = now.Add(time.Duration(skype.ExpiresIn) * time.Second)
		a.MessagingBaseURL = skype.BaseURL
		a.Region = regionFromBaseURL(skype.BaseURL)
		dirty = true
	} else if a.SpacesExpiry.Before(now.Add(60 * time.Second)) {
		tok, err := teams.RefreshAccessToken(a.RefreshToken, teams.SkypeScope)
		if err != nil {
			return nil, fmt.Errorf("refresh spaces token: %w (run: teams auth login)", err)
		}
		if tok.RefreshToken != "" {
			a.RefreshToken = tok.RefreshToken
		}
		a.SpacesToken = tok.AccessToken
		a.SpacesExpiry = now.Add(time.Duration(tok.ExpiresIn) * time.Second)
		dirty = true
	}

	if a.CSAExpiry.Before(now.Add(60 * time.Second)) {
		tok, err := teams.RefreshAccessToken(a.RefreshToken, teams.CSAScope)
		if err != nil {
			return nil, fmt.Errorf("refresh chatsvcagg token: %w (run: teams auth login)", err)
		}
		if tok.RefreshToken != "" {
			a.RefreshToken = tok.RefreshToken
		}
		a.CSAToken = tok.AccessToken
		a.CSAExpiry = now.Add(time.Duration(tok.ExpiresIn) * time.Second)
		if a.SelfMRI == "" {
			a.SelfMRI = resolveSelfMRI(a.CSAToken, a.SkypeToken)
		}
		dirty = true
	}

	if dirty {
		_ = saveAuth(a)
	}

	return teams.NewClient(a.MessagingBaseURL, a.SkypeToken, a.CSAToken, a.Region, a.SpacesToken, a.SelfMRI), nil
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via device code flow",
	RunE: func(cmd *cobra.Command, args []string) error {
		dc, err := teams.RequestDeviceCode(context.Background())
		if err != nil {
			return err
		}
		fmt.Printf("\n  Open: %s\n  Code: %s\n\n", dc.VerificationURI, dc.UserCode)

		tok, err := teams.PollDeviceCode(context.Background(), dc)
		if err != nil {
			return err
		}
		skype, err := teams.ExchangeSkypeToken(tok.AccessToken)
		if err != nil {
			return err
		}
		csa, err := teams.RefreshAccessToken(tok.RefreshToken, teams.CSAScope)
		if err != nil {
			return fmt.Errorf("mint chatsvcagg token: %w", err)
		}
		refresh := tok.RefreshToken
		if csa.RefreshToken != "" {
			refresh = csa.RefreshToken
		}
		now := time.Now()
		if err := saveAuth(&storedAuth{
			RefreshToken:     refresh,
			SpacesToken:      tok.AccessToken,
			SpacesExpiry:     now.Add(time.Duration(tok.ExpiresIn) * time.Second),
			SkypeToken:       skype.SkypeToken,
			SkypeExpiry:      now.Add(time.Duration(skype.ExpiresIn) * time.Second),
			MessagingBaseURL: skype.BaseURL,
			Region:           regionFromBaseURL(skype.BaseURL),
			CSAToken:         csa.AccessToken,
			CSAExpiry:        now.Add(time.Duration(csa.ExpiresIn) * time.Second),
			SelfMRI:          resolveSelfMRI(csa.AccessToken, skype.SkypeToken),
		}); err != nil {
			return err
		}
		fmt.Println("Successfully logged in.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := loadAuth()
		if os.IsNotExist(err) {
			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(struct {
					LoggedIn bool `json:"logged_in"`
				}{false})
			}
			fmt.Println("Not logged in.")
			return nil
		}
		if err != nil {
			return err
		}
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				LoggedIn         bool      `json:"logged_in"`
				SpacesExpiry     time.Time `json:"spaces_expiry"`
				SkypeExpiry      time.Time `json:"skype_expiry"`
				CSAExpiry        time.Time `json:"csa_expiry"`
				MessagingBaseURL string    `json:"messaging_base_url"`
				Region           string    `json:"region"`
				SelfMRI          string    `json:"self_mri"`
			}{true, a.SpacesExpiry, a.SkypeExpiry, a.CSAExpiry, a.MessagingBaseURL, a.Region, a.SelfMRI})
		}
		fmt.Printf("Spaces expiry:       %s\n", a.SpacesExpiry.Format(time.RFC3339))
		fmt.Printf("Skype expiry:        %s\n", a.SkypeExpiry.Format(time.RFC3339))
		fmt.Printf("CSA expiry:          %s\n", a.CSAExpiry.Format(time.RFC3339))
		fmt.Printf("Messaging base URL:  %s\n", a.MessagingBaseURL)
		fmt.Printf("Region:              %s\n", a.Region)
		fmt.Printf("Self MRI:            %s\n", a.SelfMRI)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := os.Remove(authPath())
		if os.IsNotExist(err) {
			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(struct {
					LoggedIn bool `json:"logged_in"`
				}{false})
			}
			fmt.Println("Not logged in.")
			return nil
		}
		if err != nil {
			return err
		}
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				LoggedOut bool `json:"logged_out"`
			}{true})
		}
		fmt.Println("Logged out.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
}
