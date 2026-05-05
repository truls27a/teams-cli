package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"teams-cli/teams"
)

type storedAuth struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	SkypeToken   string    `json:"skype_token"`
	Expiry       time.Time `json:"expiry"`
	SkypeExpiry  time.Time `json:"skype_expiry"`
	BaseURL      string    `json:"base_url"`
}

func authPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "teams-cli", "auth.json")
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

func loadClient() (*teams.Client, error) {
	a, err := loadAuth()
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("not logged in — run: teams auth login")
	}
	if err != nil {
		return nil, err
	}
	return teams.NewClient(a.BaseURL, a.SkypeToken), nil
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
		return saveAuth(&storedAuth{
			AccessToken:  tok.AccessToken,
			RefreshToken: tok.RefreshToken,
			Expiry:       time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
			SkypeToken:   skype.SkypeToken,
			SkypeExpiry:  time.Now().Add(time.Duration(skype.ExpiresIn) * time.Second),
			BaseURL:      skype.BaseURL,
		})
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := loadAuth()
		if os.IsNotExist(err) {
			fmt.Println("Not logged in.")
			return nil
		}
		if err != nil {
			return err
		}
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(a)
		}
		fmt.Printf("Token expiry: %s\n", a.Expiry.Format(time.RFC3339))
		fmt.Printf("Skype expiry: %s\n", a.SkypeExpiry.Format(time.RFC3339))
		fmt.Printf("Region:       %s\n", a.BaseURL)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := os.Remove(authPath())
		if os.IsNotExist(err) {
			fmt.Println("Not logged in.")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("Logged out.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
}
