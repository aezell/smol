package sprite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

const flyAPIBase = "https://api.fly.io"

// CLISession represents a Fly.io CLI auth session.
type CLISession struct {
	ID          string `json:"id"`
	URL         string `json:"auth_url"`
	AccessToken string `json:"access_token,omitempty"`
}

// StartLoginSession starts a new CLI auth session with Fly.io.
// Returns the auth URL the user should visit and the session ID for polling.
func StartLoginSession() (authURL string, sessionID string, err error) {
	payload, _ := json.Marshal(map[string]string{
		"name": "smol",
	})

	resp, err := http.Post(flyAPIBase+"/api/v1/cli_sessions", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", "", fmt.Errorf("starting login session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var session CLISession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", "", fmt.Errorf("decoding session: %w", err)
	}

	return session.URL, session.ID, nil
}

// WaitForLogin polls the Fly.io session endpoint until the user completes login.
// Returns the Fly.io access token.
func WaitForLogin(ctx context.Context, sessionID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	url := fmt.Sprintf("%s/api/v1/cli_sessions/%s", flyAPIBase, sessionID)

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("login timed out")
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return "", err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue // Retry on network error.
			}

			var session CLISession
			json.NewDecoder(resp.Body).Decode(&session)
			resp.Body.Close()

			if session.AccessToken != "" {
				return session.AccessToken, nil
			}
		}
	}
}

// FlyUser represents a Fly.io user.
type FlyUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// GetCurrentUser fetches the current user's info from a Fly.io token.
func GetCurrentUser(ctx context.Context, flyToken string) (*FlyUser, error) {
	query := `{"query":"query { viewer { id email } }"}`

	req, err := http.NewRequestWithContext(ctx, "POST", flyAPIBase+"/graphql", bytes.NewReader([]byte(query)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", flyAuthHeader(flyToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching user info: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Viewer FlyUser `json:"viewer"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing user info: %w", err)
	}

	if result.Data.Viewer.Email == "" {
		return nil, fmt.Errorf("could not get user info (invalid token?)")
	}

	return &result.Data.Viewer, nil
}

// FlyOrg represents a Fly.io organization.
type FlyOrg struct {
	Slug string `json:"org_slug"`
	Name string `json:"organization"`
}

// FetchOrganizations fetches the orgs available via the Fly.io token.
func FetchOrganizations(ctx context.Context, flyToken string) ([]FlyOrg, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.machines.dev/v1/tokens/current", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", flyAuthHeader(flyToken))

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching organizations: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tokens []FlyOrg `json:"tokens"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing organizations: %w", err)
	}

	// Deduplicate by slug.
	seen := map[string]bool{}
	var orgs []FlyOrg
	for _, t := range result.Tokens {
		if !seen[t.Slug] {
			seen[t.Slug] = true
			orgs = append(orgs, t)
		}
	}
	return orgs, nil
}

// CreateSpriteToken exchanges a Fly.io token for a Sprite API token.
func CreateSpriteToken(ctx context.Context, flyToken, orgSlug, apiURL string) (string, error) {
	if apiURL == "" {
		apiURL = "https://api.sprites.dev"
	}

	url := fmt.Sprintf("%s/v1/organizations/%s/tokens", apiURL, orgSlug)
	payload, _ := json.Marshal(map[string]string{
		"description": "smol CLI",
	})

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", flyAuthHeader(flyToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("creating sprite token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("token creation failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	if result.Token == "" {
		return "", fmt.Errorf("no token in response")
	}
	return result.Token, nil
}

// OpenBrowser opens a URL in the user's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func flyAuthHeader(token string) string {
	// Macaroon tokens use FlyV1, others use Bearer.
	if len(token) > 4 && (token[:5] == "fm1r_" || token[:4] == "fm2_") {
		return "FlyV1 " + token
	}
	return "Bearer " + token
}
