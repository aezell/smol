// Package sprite provides a client for the Sprite API.
package sprite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client communicates with the Sprite API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a Sprite API client from saved config (or env var override).
func NewClient() (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	token := os.Getenv("SMOL_API_TOKEN")
	if token == "" {
		token = cfg.Token
	}
	if token == "" {
		return nil, fmt.Errorf("not logged in. Run 'smol login' to get started")
	}

	baseURL := os.Getenv("SMOL_API_URL")
	if baseURL == "" {
		baseURL = cfg.APIURL
	}
	if baseURL == "" {
		baseURL = "https://api.sprites.dev"
	}

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

func (c *Client) request(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+"/v1"+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.HTTPClient.Do(req)
}

func (c *Client) doJSON(method, path string, body any, result any) error {
	resp, err := c.request(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// streamNDJSON reads an NDJSON response and calls fn for each line.
func (c *Client) streamNDJSON(resp *http.Response, fn func(line map[string]any)) error {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var obj map[string]any
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if fn != nil {
			fn(obj)
		}
	}
	return nil
}

// SpriteInfo represents a sprite instance.
type SpriteInfo struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
	URL    string `json:"url,omitempty"`
}

// CreateSprite creates a new sprite.
func (c *Client) CreateSprite(name string) error {
	resp, err := c.request("POST", "/sprites", map[string]any{
		"name": name,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// MakePublic sets the sprite's URL to be publicly accessible (no auth).
func (c *Client) MakePublic(spriteName string) error {
	resp, err := c.request("PUT", "/sprites/"+spriteName, map[string]any{
		"url_settings": map[string]string{
			"auth": "public",
		},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DestroySprite destroys a sprite.
func (c *Client) DestroySprite(name string) error {
	resp, err := c.request("DELETE", "/sprites/"+name, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListSprites lists all sprites.
func (c *Client) ListSprites() ([]SpriteInfo, error) {
	var result struct {
		Sprites []SpriteInfo `json:"sprites"`
	}
	if err := c.doJSON("GET", "/sprites", nil, &result); err != nil {
		return nil, err
	}
	return result.Sprites, nil
}

// GetSprite gets info about a specific sprite.
func (c *Client) GetSprite(name string) (*SpriteInfo, error) {
	var info SpriteInfo
	if err := c.doJSON("GET", "/sprites/"+name, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// WaitReady polls the sprite until exec succeeds (sprite is awake).
func (c *Client) WaitReady(spriteName string) error {
	for i := 0; i < 30; i++ {
		_, err := c.Exec(spriteName, "true")
		if err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("sprite %q did not become ready", spriteName)
}

// Exec runs a shell command on a sprite and returns combined output.
func (c *Client) Exec(spriteName string, command string) (string, error) {
	// The exec API takes command args as query params, not JSON.
	// We wrap in bash -c to support shell features (pipes, &&, etc).
	url := c.BaseURL + "/v1/sprites/" + spriteName + "/exec"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/octet-stream")

	q := req.URL.Query()
	q.Add("cmd", "bash")
	q.Add("cmd", "-c")
	q.Add("cmd", command)
	q.Set("path", "bash")
	req.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("exec error %d: %s", resp.StatusCode, string(body))
	}

	return strings.TrimSpace(string(body)), nil
}

// UploadTar uploads a tar stream to a sprite at the given destination.
func (c *Client) UploadTar(spriteName string, tarData io.Reader, destDir string) error {
	url := c.BaseURL + "/v1/sprites/" + spriteName + "/exec"
	req, err := http.NewRequest("POST", url, tarData)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/octet-stream")

	q := req.URL.Query()
	q.Add("cmd", "tar")
	q.Add("cmd", "xzf")
	q.Add("cmd", "-")
	q.Add("cmd", "-C")
	q.Add("cmd", destDir)
	q.Set("path", "tar")
	q.Set("stdin", "true")
	req.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// Service represents a service running on a sprite.
type Service struct {
	Name  string        `json:"name"`
	Cmd   string        `json:"cmd,omitempty"`
	Args  []string      `json:"args,omitempty"`
	State *ServiceState `json:"state,omitempty"`
}

// ServiceState represents the runtime state of a service.
type ServiceState struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	PID    int    `json:"pid,omitempty"`
}

// CreateService creates or updates a service on a sprite.
func (c *Client) CreateService(spriteName, serviceName, cmd string, args []string, httpPort int) error {
	resp, err := c.request("PUT", "/sprites/"+spriteName+"/services/"+serviceName, map[string]any{
		"cmd":       cmd,
		"args":      args,
		"http_port": httpPort,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	// Drain the streaming response.
	io.Copy(io.Discard, resp.Body)
	return nil
}

// StartService starts a service on a sprite.
func (c *Client) StartService(spriteName, serviceName string) error {
	resp, err := c.request("POST", "/sprites/"+spriteName+"/services/"+serviceName+"/start", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

// StopService stops a service on a sprite.
func (c *Client) StopService(spriteName, serviceName string) error {
	resp, err := c.request("POST", "/sprites/"+spriteName+"/services/"+serviceName+"/stop", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

// DeleteService deletes a service from a sprite. Ignores 404.
func (c *Client) DeleteService(spriteName, serviceName string) error {
	resp, err := c.request("DELETE", "/sprites/"+spriteName+"/services/"+serviceName, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListServices lists services on a sprite.
func (c *Client) ListServices(spriteName string) ([]Service, error) {
	var services []Service
	if err := c.doJSON("GET", "/sprites/"+spriteName+"/services", nil, &services); err != nil {
		return nil, err
	}
	return services, nil
}

// ServiceLogs gets logs for a service.
func (c *Client) ServiceLogs(spriteName, serviceName string, lines int) (string, error) {
	path := fmt.Sprintf("/sprites/%s/services/%s/logs?lines=%d", spriteName, serviceName, lines)
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
