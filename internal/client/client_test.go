package client

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theinventor/momentumticker-cli/internal/config"
)

func TestEnvWinsOverDefaultProfile(t *testing.T) {
	t.Setenv(EnvToken, "env-token-long")
	t.Setenv(EnvURL, "https://env.example")
	t.Setenv("MOMENTUMTICKER_CONFIG", filepath.Join(t.TempDir(), "config.json"))

	f := &config.File{}
	f.Put("dev", config.Profile{APIURL: "https://config.example", APIToken: "config-token-long"})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	c := New()
	if c.Token != "env-token-long" {
		t.Fatalf("token = %q, want env token", c.Token)
	}
	if c.BaseURL != "https://env.example" {
		t.Fatalf("baseURL = %q, want env URL", c.BaseURL)
	}
	if c.Source != "env" {
		t.Fatalf("source = %q, want env", c.Source)
	}
}

func TestDoJSONSummarizesHTMLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><div class="message">Market data request failed: 404</div><pre>very long trace</pre></body></html>`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.DoJSON(http.MethodGet, "/api/v1/report", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	text := err.Error()
	if !strings.Contains(text, "HTML error response: Market data request failed: 404") {
		t.Fatalf("unexpected error: %s", text)
	}
	if strings.Contains(text, "<html>") || strings.Contains(text, "very long trace") {
		t.Fatalf("HTML body was not summarized: %s", text)
	}
}

func TestExplicitProfileDoesNotFallBackToEnvToken(t *testing.T) {
	t.Setenv(EnvToken, "env-token-long")
	t.Setenv(EnvURL, "https://env.example")
	t.Setenv("MOMENTUMTICKER_CONFIG", filepath.Join(t.TempDir(), "config.json"))

	f := &config.File{}
	f.Put("dev", config.Profile{APIURL: "https://config.example", APIToken: "config-token-long"})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	c := NewWithProfile("missing")
	if c.Token != "" {
		t.Fatalf("missing explicit profile used fallback token %q", c.Token)
	}
	if c.BaseURL != "https://env.example" {
		t.Fatalf("missing profile should still honor env URL for diagnostics, got %q", c.BaseURL)
	}
}
