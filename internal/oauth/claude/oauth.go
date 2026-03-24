package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/oauth"
)

const (
	clientID    = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	redirectURI = "https://platform.claude.com/oauth/code/callback"
	tokenURL    = "https://platform.claude.com/v1/oauth/token"
)

// AuthorizeURL returns the Claude Code Max OAuth2 authorization URL.
func AuthorizeURL(verifier, challenge string) (string, error) {
	u, err := url.Parse("https://claude.ai/oauth/authorize")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", "user:profile user:inference user:sessions:claude_code user:mcp_servers")
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", verifier)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ExchangeToken exchanges the authorization code for an OAuth2 token.
func ExchangeToken(ctx context.Context, code, verifier string) (*oauth.Token, error) {
	code = strings.TrimSpace(code)
	// Strip state suffix if pasted as "code#state".
	if i := strings.IndexByte(code, '#'); i >= 0 {
		code = code[:i]
	}

	reqBody := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     clientID,
		"redirect_uri":  redirectURI,
		"code_verifier": verifier,
	}

	resp, err := request(ctx, "POST", tokenURL, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("claude code max: failed to exchange token: status %d body %q", resp.StatusCode, string(body))
	}

	var token oauth.Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.SetExpiresAt()
	return &token, nil
}

// RefreshToken refreshes the OAuth2 token using the provided refresh token.
func RefreshToken(ctx context.Context, refreshToken string) (*oauth.Token, error) {
	reqBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     clientID,
	}

	resp, err := request(ctx, "POST", tokenURL, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("claude code max: failed to refresh token: status %d body %q", resp.StatusCode, string(body))
	}

	var token oauth.Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	token.SetExpiresAt()
	return &token, nil
}

// RefreshViaCLI triggers the Claude Code CLI to refresh its own token, then
// re-reads the updated credentials from disk.
func RefreshViaCLI() (*oauth.Token, bool) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "-p", ".", "--model", "haiku")
	cmd.Env = append(os.Environ(), "TERM=dumb")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()

	return TokenFromDisk()
}

// TokenFromDisk reads Claude Code credentials from ~/.claude/.credentials.json.
func TokenFromDisk() (*oauth.Token, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, false
	}

	data, err := os.ReadFile(home + "/.claude/.credentials.json")
	if err != nil {
		return nil, false
	}

	var creds struct {
		ClaudeAiOauth *struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresAt    int64  `json:"expiresAt"` // milliseconds
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &creds); err != nil || creds.ClaudeAiOauth == nil {
		return nil, false
	}

	c := creds.ClaudeAiOauth
	if c.AccessToken == "" {
		return nil, false
	}

	token := &oauth.Token{
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		ExpiresAt:    c.ExpiresAt / 1000, // ms → seconds
	}
	token.SetExpiresIn()
	return token, true
}

func request(ctx context.Context, method, url string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "anthropic")

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}
