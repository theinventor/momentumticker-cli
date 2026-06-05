package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theinventor/momentumticker-cli/internal/client"
	"github.com/theinventor/momentumticker-cli/internal/config"
	"github.com/theinventor/momentumticker-cli/internal/credstore"
)

func newAuthCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage saved MomentumTicker API token profiles",
		Long: `Manage normal-user MomentumTicker API token profiles.

Resolution order for normal commands:
  1. --profile NAME, when passed
  2. MOMENTUMTICKER_TOKEN and MOMENTUMTICKER_URL
  3. the saved default profile

New profiles use the OS keychain when available. Pass --storage=file on
headless servers or CI to keep the token in the mode-0600 config file.`,
	}
	c.AddCommand(newAuthSaveCmd())
	c.AddCommand(newAuthStatusCmd())
	c.AddCommand(newAuthListCmd())
	c.AddCommand(newAuthUseCmd())
	c.AddCommand(newAuthLogoutCmd())
	c.AddCommand(newAuthMigrateCmd())
	return c
}

func newAuthSaveCmd() *cobra.Command {
	var profileName, token, baseURL, storage string
	c := &cobra.Command{
		Use:   "save",
		Short: "Save an existing normal-user API token as a profile",
		Example: `  momentum auth save --profile dev --base http://127.0.0.1:3007 --token "$MOMENTUMTICKER_TOKEN"
  momentum auth save --profile prod --token "$MOMENTUMTICKER_TOKEN" --storage=file`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profileName == "" {
				return fmt.Errorf("--profile is required")
			}
			if token == "" {
				return fmt.Errorf("--token is required")
			}
			if baseURL == "" {
				baseURL = client.DefaultAPIURL
			}
			backend, err := credstore.ResolveBackend(storage)
			if err != nil {
				return err
			}
			canon, err := persistProfile(profileName, config.Profile{APIURL: baseURL}, token, backend)
			if err != nil {
				return err
			}
			f, _ := config.Load()
			fmt.Fprintf(cmd.OutOrStdout(), "saved profile %q (storage: %s)\n", profileName, credstore.Describe(canon))
			fmt.Fprintf(cmd.OutOrStdout(), "config: %s\n", config.Path())
			if f != nil && f.DefaultProfile == profileName {
				fmt.Fprintln(cmd.OutOrStdout(), "set as default profile")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "token: %s\n", maskToken(token))
			return nil
		},
	}
	c.Flags().StringVar(&profileName, "profile", "default", "profile name")
	c.Flags().StringVar(&token, "token", "", "normal-user API token")
	c.Flags().StringVar(&baseURL, "base", client.DefaultAPIURL, "MomentumTicker base URL")
	c.Flags().StringVar(&storage, "storage", "", "token storage: auto (default), keychain, or file")
	return c
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [PROFILE]",
		Short: "Show the active or named profile without printing the token",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				name = f.DefaultProfile
			}
			p, ok := f.Get(name)
			if !ok || name == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "no active profile")
				return nil
			}
			secret, _ := credstore.Get(name, p.Backend, p.APIToken)
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), map[string]any{
					"profile": name,
					"base":    p.APIURL,
					"storage": storageName(p.Backend),
					"token":   maskToken(secret),
					"active":  name == f.DefaultProfile,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "profile: %s\nbase: %s\nstorage: %s\ntoken: %s\n", name, p.APIURL, storageName(p.Backend), maskToken(secret))
			return nil
		},
	}
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), redactedAuthList(f))
			}
			for _, name := range f.Names() {
				active := ""
				if name == f.DefaultProfile {
					active = " *"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s%s\n", name, f.Profiles[name].APIURL, active)
			}
			return nil
		},
	}
}

func redactedAuthList(f *config.File) map[string]any {
	profiles := make([]map[string]any, 0, len(f.Profiles))
	for _, name := range f.Names() {
		p := f.Profiles[name]
		profiles = append(profiles, map[string]any{
			"name":       name,
			"api_url":    p.APIURL,
			"storage":    storageName(p.Backend),
			"api_token":  maskToken(p.APIToken),
			"is_default": name == f.DefaultProfile,
			"created_at": p.CreatedAt,
		})
	}
	return map[string]any{
		"config_path":     config.Path(),
		"default_profile": f.DefaultProfile,
		"profiles":        profiles,
	}
}

func newAuthUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use PROFILE",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			if err := f.SetDefault(args[0]); err != nil {
				return err
			}
			if err := f.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "active profile: %s\n", args[0])
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout [PROFILE]",
		Short: "Remove a saved profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			name := f.DefaultProfile
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" || !f.Delete(name) {
				return fmt.Errorf("profile not found")
			}
			_ = credstore.Delete(name)
			if err := f.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed profile %s\n", name)
			return nil
		},
	}
}

func newAuthMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate [PROFILE]",
		Short: "Move file-backed profile tokens into the OS keychain",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			names := f.Names()
			if len(args) > 0 {
				names = args
			}
			for _, name := range names {
				p, ok := f.Profiles[name]
				if !ok || p.APIToken == "" {
					continue
				}
				canon, err := persistProfile(name, p, p.APIToken, credstore.BackendKeychain)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "migrated %s to %s\n", name, credstore.Describe(canon))
			}
			return nil
		},
	}
}

func persistProfile(name string, p config.Profile, secret, backend string) (string, error) {
	canon, err := credstore.Put(name, backend, secret)
	if err != nil {
		return "", err
	}
	if canon == credstore.BackendKeychain {
		p.APIToken = ""
	} else {
		p.APIToken = secret
	}
	p.Backend = canon
	f, err := config.Load()
	if err != nil {
		return "", err
	}
	f.Put(name, p)
	if err := f.Save(); err != nil {
		return "", err
	}
	return canon, nil
}

func storageName(backend string) string {
	if backend == "" {
		return credstore.BackendFile
	}
	return backend
}

func maskToken(token string) string {
	if token == "" {
		return "(none)"
	}
	if len(token) < 12 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}
