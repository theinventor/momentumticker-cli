package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theinventor/momentumticker-cli/internal/credstore"
)

func runMomentumCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := NewRootCmd()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errOut.String(), err
}

func TestRootHelpSurfacesMonsterMailboxGradeCommands(t *testing.T) {
	stdout, _, err := runMomentumCmd(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"normal-user CLI",
		"auth",
		"doctor",
		"folio",
		"research",
		"update",
		"--profile",
		"--version",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help missing %q:\n%s", want, stdout)
		}
	}
}

func TestUpdateHelpDocumentsChecksumsAndCache(t *testing.T) {
	stdout, _, err := runMomentumCmd(t, "update", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"checksums.txt", "--check", "--no-cache", "--to"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("update help missing %q:\n%s", want, stdout)
		}
	}
}

func TestAuthSaveStatusMasksToken(t *testing.T) {
	t.Setenv("MOMENTUMTICKER_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("MOMENTUMTICKER_DISABLE_KEYCHAIN", "1")

	stdout, _, err := runMomentumCmd(t, "auth", "save",
		"--profile", "dev",
		"--base", "http://127.0.0.1:3007",
		"--token", "mt_test_secret_token",
		"--storage", "file",
	)
	if err != nil {
		t.Fatalf("auth save: %v", err)
	}
	if strings.Contains(stdout, "mt_test_secret_token") {
		t.Fatalf("save printed raw token:\n%s", stdout)
	}

	stdout, _, err = runMomentumCmd(t, "auth", "status", "dev")
	if err != nil {
		t.Fatalf("auth status: %v", err)
	}
	if !strings.Contains(stdout, "mt_tes...oken") {
		t.Fatalf("status did not print token fingerprint:\n%s", stdout)
	}
	if strings.Contains(stdout, "mt_test_secret_token") {
		t.Fatalf("status printed raw token:\n%s", stdout)
	}
}

func TestFolioCreateRequestShape(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42,"name":"Semis","display_name":"Semis","kind":"research","access":"owner"}`))
	}))
	defer srv.Close()

	t.Setenv("MOMENTUMTICKER_TOKEN", "api-token")
	t.Setenv("MOMENTUMTICKER_URL", srv.URL)

	stdout, _, err := runMomentumCmd(t, "folio", "create", "--name", "Semis", "--description", "Chip scan", "--entries", "NVDA 2\nAMD 3")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/folios" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if gotAuth != "Bearer api-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotBody["name"] != "Semis" || gotBody["description"] != "Chip scan" || gotBody["entries"] != "NVDA 2\nAMD 3" {
		t.Fatalf("body = %#v", gotBody)
	}
	if !strings.Contains(stdout, "created folio 42: Semis") {
		t.Fatalf("stdout = %s", stdout)
	}
}

func TestShareAcceptRequestShape(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":7,"status":"accepted","permission":"read"}`))
	}))
	defer srv.Close()

	t.Setenv("MOMENTUMTICKER_TOKEN", "api-token")
	t.Setenv("MOMENTUMTICKER_URL", srv.URL)

	stdout, _, err := runMomentumCmd(t, "folio", "share", "accept", "7")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/api/v1/folio_shares/7" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if gotBody["status"] != "accepted" {
		t.Fatalf("body = %#v", gotBody)
	}
	if !strings.Contains(stdout, "share 7 accepted") {
		t.Fatalf("stdout = %s", stdout)
	}
}

func TestWhoamiUsesProfileEndpointAndNeverRawToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/profile" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer api-token-secret" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"email":"kevin@example.com","folios_count":3}`))
	}))
	defer srv.Close()

	t.Setenv("MOMENTUMTICKER_TOKEN", "api-token-secret")
	t.Setenv("MOMENTUMTICKER_URL", srv.URL)
	t.Setenv("MOMENTUMTICKER_UPDATE_CACHE", filepath.Join(t.TempDir(), "cache.json"))

	stdout, _, err := runMomentumCmd(t, "whoami")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"email": "kevin@example.com"`) {
		t.Fatalf("whoami output missing profile:\n%s", stdout)
	}
	if strings.Contains(stdout, "api-token-secret") {
		t.Fatalf("whoami leaked raw token:\n%s", stdout)
	}
}

func TestWorkflowCommandRequestShapes(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/report":
			_, _ = w.Write([]byte(`{"folio":{"name":"Smoke","display_name":"Smoke","access":"owner"},"results":[]}`))
		case "/api/v1/research_runs":
			if r.Method == http.MethodPost {
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["tickers_text"] != "NVDA\nAMD" {
					t.Fatalf("research body = %#v", body)
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":9,"name":"Semis","status":"queued","tickers":["NVDA","AMD"]}`))
				return
			}
			_, _ = w.Write([]byte(`{"research_runs":[{"id":9,"name":"Semis","status":"queued","tickers":["NVDA","AMD"]}]}`))
		case "/api/v1/report_schedule":
			if r.Method == http.MethodPatch {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["recipient_email"] != "kevin@example.com" {
					t.Fatalf("schedule body = %#v", body)
				}
			}
			_, _ = w.Write([]byte(`{"recipient_email":"kevin@example.com","enabled":true}`))
		case "/api/v1/report_emails":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["recipient_email"] != "kevin@example.com" {
				t.Fatalf("email body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ok":true,"recipient_email":"kevin@example.com"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.RequestURI())
		}
	}))
	defer srv.Close()

	t.Setenv("MOMENTUMTICKER_TOKEN", "api-token")
	t.Setenv("MOMENTUMTICKER_URL", srv.URL)

	commands := [][]string{
		{"report", "--folio", "master"},
		{"research", "run", "--name", "Semis", "--tickers", "NVDA,AMD"},
		{"research", "list"},
		{"schedule", "set", "--email", "kevin@example.com", "--enabled"},
		{"email", "kevin@example.com"},
	}
	for _, args := range commands {
		if _, _, err := runMomentumCmd(t, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	want := []string{
		"GET /api/v1/report?folio_id=master",
		"POST /api/v1/research_runs",
		"GET /api/v1/research_runs",
		"PATCH /api/v1/report_schedule",
		"POST /api/v1/report_emails",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("seen requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestAuthSaveCanUseMockKeychain(t *testing.T) {
	t.Setenv("MOMENTUMTICKER_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	_, restore := credstore.UseMockKeyring()
	defer restore()

	_, _, err := runMomentumCmd(t, "auth", "save",
		"--profile", "kc",
		"--base", "http://127.0.0.1:3007",
		"--token", "mt_keychain_secret",
		"--storage", "keychain",
	)
	if err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runMomentumCmd(t, "auth", "status", "kc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "storage: keychain") {
		t.Fatalf("status missing keychain storage:\n%s", stdout)
	}
}
