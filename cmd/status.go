package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theinventor/momentumticker-cli/internal/config"
	"github.com/theinventor/momentumticker-cli/internal/updater"
)

var osRemove = os.Remove
var osExecutable = os.Executable

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the current MomentumTicker API identity and update hint",
		Long: `Calls GET /api/v1/profile with the resolved normal-user API token.

The output never prints the raw token. It includes a cached, non-fatal update
hint when a newer public release exists.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := requireAPIClient()
			if err != nil {
				return err
			}
			var profile map[string]any
			if err := c.DoJSON(http.MethodGet, "/api/v1/profile", nil, nil, &profile); err != nil {
				return err
			}
			out := map[string]any{
				"api_url":     c.BaseURL,
				"token":       c.MaskedToken(),
				"source":      nullableSource(c.Source),
				"cli_version": Version,
				"profile":     profile,
			}
			if name, ok := strings.CutPrefix(c.Source, "profile:"); ok {
				out["profile_name"] = name
			}
			info := updater.CheckForUpdate(Version)
			if info.Available {
				out["update_available"] = map[string]any{
					"current": info.Current,
					"latest":  info.Latest,
					"url":     info.URL,
					"hint":    "run `momentum update` to install",
				}
			}
			return printJSON(cmd.OutOrStdout(), out)
		},
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Show local CLI configuration, auth source, and update status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := newAPIClient()
			f, _ := config.Load()
			out := map[string]any{
				"config_path":   config.Path(),
				"default":       "",
				"profile_count": 0,
				"api_url":       c.BaseURL,
				"token":         c.MaskedToken(),
				"source":        nullableSource(c.Source),
				"cli_version":   Version,
				"update":        updater.CheckForUpdate(Version),
			}
			if f != nil {
				out["default"] = f.DefaultProfile
				out["profile_count"] = len(f.Profiles)
			}
			return printJSON(cmd.OutOrStdout(), out)
		},
	}
}

func nullableSource(source string) any {
	if source == "" {
		return nil
	}
	return source
}

func newUpdateCmd() *cobra.Command {
	var checkOnly, noCache bool
	var pinTo string
	c := &cobra.Command{
		Use:   "update",
		Short: "Check for and install a newer momentum binary",
		Long: `Updates the momentum binary in place.

Default: check the GitHub Releases API, download the asset for this platform,
verify it against checksums.txt, and atomically replace the running binary.

  momentum update
  momentum update --check
  momentum update --check --no-cache
  momentum update --to v0.4.0`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if checkOnly {
				if noCache {
					_ = osRemove(updater.CachePath())
				}
				info := updater.CheckForUpdate(Version)
				if err := printJSON(cmd.OutOrStdout(), info); err != nil {
					return err
				}
				if info.Available {
					return fmt.Errorf("update available: %s", info.Latest)
				}
				return nil
			}
			rel, err := updater.LatestRelease(nil)
			if err != nil {
				return fmt.Errorf("could not fetch release info: %w", err)
			}
			if pinTo != "" && !strings.EqualFold(pinTo, rel.TagName) {
				return fmt.Errorf("--to %s differs from latest %s; pinning older releases is not implemented yet", pinTo, rel.TagName)
			}
			exe, err := osExecutable()
			if err != nil {
				return fmt.Errorf("locate own binary: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "current: %s\nlatest:  %s\ntarget:  %s\ndownloading...\n", Version, rel.TagName, exe)
			if err := updater.Install(rel, exe, nil); err != nil {
				return fmt.Errorf("install: %w", err)
			}
			_ = osRemove(updater.CachePath())
			fmt.Fprintf(cmd.OutOrStdout(), "installed %s; run `momentum --version` to verify\n", rel.TagName)
			return nil
		},
	}
	c.Flags().BoolVar(&checkOnly, "check", false, "only check; return an error if an update is available")
	c.Flags().BoolVar(&noCache, "no-cache", false, "ignore the 24h update cache")
	c.Flags().StringVar(&pinTo, "to", "", "install a specific tag when it is the latest release")
	return c
}
