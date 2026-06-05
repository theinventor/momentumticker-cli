// Package config reads and writes MomentumTicker CLI profiles.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Profile struct {
	APIURL    string `json:"api_url"`
	APIToken  string `json:"api_token,omitempty"`
	Backend   string `json:"backend,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type File struct {
	DefaultProfile string             `json:"default_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles,omitempty"`
}

func Path() string {
	if explicit := os.Getenv("MOMENTUMTICKER_CONFIG"); explicit != "" {
		return explicit
	}
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "momentumticker", "config.json")
}

func Load() (*File, error) {
	raw, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Profiles: map[string]Profile{}}, nil
		}
		return nil, fmt.Errorf("read %s: %w", Path(), err)
	}
	var f File
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", Path(), err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	return &f, nil
}

func (f *File) Save() error {
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir config dir: %w", err)
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return fmt.Errorf("chmod %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}

func (f *File) Get(name string) (*Profile, bool) {
	if name == "" {
		name = f.DefaultProfile
	}
	if name == "" {
		return nil, false
	}
	p, ok := f.Profiles[name]
	if !ok {
		return nil, false
	}
	return &p, true
}

func (f *File) Put(name string, p Profile) {
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	if p.CreatedAt == "" {
		p.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	f.Profiles[name] = p
	if f.DefaultProfile == "" {
		f.DefaultProfile = name
	}
}

func (f *File) Delete(name string) bool {
	if _, ok := f.Profiles[name]; !ok {
		return false
	}
	delete(f.Profiles, name)
	if f.DefaultProfile == name {
		f.DefaultProfile = ""
		if len(f.Profiles) > 0 {
			names := f.Names()
			f.DefaultProfile = names[0]
		}
	}
	return true
}

func (f *File) SetDefault(name string) error {
	if _, ok := f.Profiles[name]; !ok {
		return fmt.Errorf("no profile named %q", name)
	}
	f.DefaultProfile = name
	return nil
}

func (f *File) Names() []string {
	names := make([]string, 0, len(f.Profiles))
	for name := range f.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
