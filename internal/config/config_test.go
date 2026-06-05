package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveWritesMode0600AndRoundTripsProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("MOMENTUMTICKER_CONFIG", path)

	f := &File{}
	f.Put("dev", Profile{APIURL: "http://127.0.0.1:3007", APIToken: "secret-token", Backend: "file"})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed File
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.DefaultProfile != "dev" {
		t.Fatalf("default profile = %q, want dev", parsed.DefaultProfile)
	}
	if parsed.Profiles["dev"].APIToken != "secret-token" {
		t.Fatalf("token did not round trip")
	}
}
