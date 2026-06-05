// Package client wraps the normal-user MomentumTicker HTTP API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/theinventor/momentumticker-cli/internal/config"
	"github.com/theinventor/momentumticker-cli/internal/credstore"
)

const (
	EnvToken        = "MOMENTUMTICKER_TOKEN"
	EnvURL          = "MOMENTUMTICKER_URL"
	DefaultAPIURL   = "https://momentumticker.com"
	AuthScheme      = "Bearer"
	UserAgentPrefix = "momentum-cli"
)

type Options struct {
	Profile string
	BaseURL string
	Token   string
	Version string
}

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Version    string
	Source     string
	Backend    string
}

type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("%s %s failed: HTTP %d", e.Method, e.Path, e.StatusCode)
	}
	return fmt.Sprintf("%s %s failed: HTTP %d: %s", e.Method, e.Path, e.StatusCode, body)
}

func New() *Client {
	return NewWithOptions(Options{})
}

func NewWithProfile(profile string) *Client {
	return NewWithOptions(Options{Profile: profile})
}

func NewWithOptions(opts Options) *Client {
	c := &Client{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Version:    opts.Version,
	}

	if opts.Profile != "" {
		if f, err := config.Load(); err == nil {
			if p, ok := f.Get(opts.Profile); ok {
				c.loadFromProfile(opts.Profile, p)
			}
		}
	} else if envToken := os.Getenv(EnvToken); envToken != "" {
		c.Token = envToken
		c.Source = "env"
		c.Backend = credstore.BackendEnv
		c.BaseURL = strings.TrimRight(os.Getenv(EnvURL), "/")
	} else if f, err := config.Load(); err == nil {
		if p, ok := f.Get(""); ok {
			c.loadFromProfile(f.DefaultProfile, p)
		}
	}

	if opts.BaseURL != "" {
		c.BaseURL = strings.TrimRight(opts.BaseURL, "/")
	}
	if opts.Token != "" {
		c.Token = opts.Token
		c.Source = "flag"
		c.Backend = credstore.BackendEnv
	}
	if c.BaseURL == "" {
		if envURL := os.Getenv(EnvURL); envURL != "" {
			c.BaseURL = strings.TrimRight(envURL, "/")
		}
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultAPIURL
	}
	return c
}

func (c *Client) loadFromProfile(name string, p *config.Profile) {
	c.BaseURL = strings.TrimRight(p.APIURL, "/")
	c.Source = "profile:" + name
	if p.Backend == "" {
		c.Backend = credstore.BackendFile
	} else {
		c.Backend = p.Backend
	}
	if secret, err := credstore.Get(name, p.Backend, p.APIToken); err == nil {
		c.Token = secret
	}
}

func (c *Client) MaskedToken() string {
	if c.Token == "" {
		return "(none)"
	}
	if len(c.Token) < 12 {
		return "***"
	}
	return c.Token[:6] + "..." + c.Token[len(c.Token)-4:]
}

func (c *Client) Do(method, path string, body any, query url.Values) (*http.Response, error) {
	u := c.BaseURL + path
	if query != nil && len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgentPrefix+"/"+c.Version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", AuthScheme+" "+c.Token)
	}
	return c.HTTPClient.Do(req)
}

func (c *Client) DoJSON(method, path string, body any, query url.Values, out any) error {
	resp, err := c.Do(method, path, body, query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return APIError{Method: method, Path: path, StatusCode: resp.StatusCode, Body: summarizeErrorBody(string(data))}
	}
	if out != nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func summarizeErrorBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	lower := strings.ToLower(body)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") {
		if msg := between(body, `<div class="message">`, "</div>"); msg != "" {
			return "HTML error response: " + compactText(stripTags(msg))
		}
		if title := between(strings.ToLower(body), "<title>", "</title>"); title != "" {
			return "HTML error response: " + compactText(stripTags(title))
		}
		return "HTML error response"
	}
	return truncate(compactText(body), 800)
}

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	i += len(start)
	j := strings.Index(s[i:], end)
	if j < 0 {
		return ""
	}
	return s[i : i+j]
}

func stripTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				out.WriteRune(r)
			}
		}
	}
	return html.UnescapeString(out.String())
}

func compactText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
