package kiwify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://public-api.kiwify.com/v1"

// TokenStore persists OAuth access tokens outside this package (e.g. encrypted DB).
type TokenStore interface {
	LoadToken() (token string, expiresAt time.Time, err error)
	SaveToken(token string, expiresAt time.Time) error
}

// Config configures the Kiwify Public API client.
type Config struct {
	BaseURL      string // default https://public-api.kiwify.com/v1
	AccountID    string
	ClientID     string
	ClientSecret string
	TokenStore   TokenStore
	HTTPClient   *http.Client
}

// Client talks to the Kiwify Public API with OAuth token caching.
type Client struct {
	baseURL      string
	accountID    string
	clientID     string
	clientSecret string
	store        TokenStore
	http         *http.Client
}

// NewClient builds a Client from cfg. TokenStore must be non-nil for GetToken/do.
func NewClient(cfg Config) *Client {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{
		baseURL:      base,
		accountID:    cfg.AccountID,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		store:        cfg.TokenStore,
		http:         hc,
	}
}

// GetToken returns a valid bearer token, refreshing via OAuth when needed.
// A stored token is reused only if it remains valid for more than 60 seconds.
func (c *Client) GetToken(ctx context.Context) (string, error) {
	if c.store == nil {
		return "", fmt.Errorf("kiwify: TokenStore is required")
	}
	token, exp, err := c.store.LoadToken()
	if err != nil {
		return "", fmt.Errorf("kiwify: load token: %w", err)
	}
	if token != "" && exp.After(time.Now().Add(60*time.Second)) {
		return token, nil
	}
	return c.refreshToken(ctx)
}

func (c *Client) refreshToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("kiwify: oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("kiwify: oauth request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("kiwify: read oauth response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", c.apiErrorFromResponse(res.StatusCode, body)
	}

	var tr oauthTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("kiwify: decode oauth response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("kiwify: oauth response missing access_token")
	}
	secs, err := parseExpiresIn(tr.ExpiresIn)
	if err != nil {
		return "", fmt.Errorf("kiwify: expires_in: %w", err)
	}
	expiresAt := time.Now().Add(time.Duration(secs) * time.Second)
	if err := c.store.SaveToken(tr.AccessToken, expiresAt); err != nil {
		return "", fmt.Errorf("kiwify: save token: %w", err)
	}
	return tr.AccessToken, nil
}

func parseExpiresIn(v any) (int64, error) {
	switch x := v.(type) {
	case nil:
		return 3600, nil // sensible default if omitted
	case float64:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	case string:
		if x == "" {
			return 3600, nil
		}
		return strconv.ParseInt(x, 10, 64)
	case int:
		return int64(x), nil
	case int64:
		return x, nil
	default:
		return 0, fmt.Errorf("unexpected type %T", v)
	}
}

// GetJSON performs an authenticated GET and decodes JSON into out (optional).
func (c *Client) GetJSON(ctx context.Context, path string, query map[string]string, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, out)
}

// PostJSON performs an authenticated POST with a JSON body.
func (c *Client) PostJSON(ctx context.Context, path string, query map[string]string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, query, body, out)
}

// PutJSON performs an authenticated PUT with a JSON body.
func (c *Client) PutJSON(ctx context.Context, path string, query map[string]string, body any, out any) error {
	return c.do(ctx, http.MethodPut, path, query, body, out)
}

// DeleteJSON performs an authenticated DELETE.
func (c *Client) DeleteJSON(ctx context.Context, path string, query map[string]string, out any) error {
	return c.do(ctx, http.MethodDelete, path, query, nil, out)
}

func (c *Client) do(ctx context.Context, method, path string, query map[string]string, body any, out any) error {
	return c.doOnce(ctx, method, path, query, body, out, true)
}

func (c *Client) doOnce(ctx context.Context, method, path string, query map[string]string, body any, out any, retryOn401 bool) error {
	token, err := c.GetToken(ctx)
	if err != nil {
		return err
	}

	u, err := c.buildURL(path, query)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("kiwify: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("kiwify: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-kiwify-account-id", c.accountID)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("kiwify: request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("kiwify: read response: %w", err)
	}

	if res.StatusCode == http.StatusUnauthorized && retryOn401 {
		// Invalidate cached token and refresh once.
		if c.store != nil {
			_ = c.store.SaveToken("", time.Time{})
		}
		if _, err := c.refreshToken(ctx); err != nil {
			return err
		}
		return c.doOnce(ctx, method, path, query, body, out, false)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return c.apiErrorFromResponse(res.StatusCode, respBody)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("kiwify: decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) buildURL(path string, query map[string]string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	raw := c.baseURL + path
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("kiwify: parse url: %w", err)
	}
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func (c *Client) apiErrorFromResponse(status int, body []byte) *APIError {
	var parsed apiErrorBody
	_ = json.Unmarshal(body, &parsed)
	msg := parsed.Message
	if msg == "" {
		msg = parsed.Error
	}
	return &APIError{
		Status:      status,
		Code:        parsed.Code,
		Message:     msg,
		UserMessage: userMessageForStatus(status, msg),
		Body:        string(body),
	}
}
