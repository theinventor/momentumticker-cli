package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/theinventor/momentumticker-cli/internal/client"
)

var Version = "dev"

var rootProfile string
var rootBaseURL string
var rootToken string
var rootJSON bool

func init() {
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	v := info.Main.Version
	if v != "" && v != "(devel)" {
		Version = v
	}
}

func NewRootCmd() *cobra.Command {
	rootProfile = ""
	rootBaseURL = ""
	rootToken = ""
	rootJSON = false

	root := &cobra.Command{
		Use:   "momentum",
		Short: "MomentumTicker CLI for normal-user Folio workflows",
		Long: `momentum is the public normal-user CLI for MomentumTicker.

It uses normal MomentumTicker API tokens only. It does not contain Super Admin
commands or /super-admin API access.`,
		Example: `  momentum auth save --profile dev --base http://127.0.0.1:3007 --token "$MOMENTUMTICKER_TOKEN"
  momentum --profile dev whoami
  momentum folio list
  momentum folio holdings set 42 < entries.txt
  momentum update --check`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&rootProfile, "profile", "", "saved auth profile to use")
	root.PersistentFlags().StringVar(&rootBaseURL, "base", "", "MomentumTicker base URL; overrides profile URL")
	root.PersistentFlags().StringVar(&rootToken, "token", "", "API token; overrides profile token")
	root.PersistentFlags().BoolVar(&rootJSON, "json", false, "emit JSON where the command supports structured output")

	root.AddCommand(newAuthCmd())
	root.AddCommand(newWhoamiCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newFolioCmd())
	root.AddCommand(newIndexCmd())
	root.AddCommand(newReportCmd())
	root.AddCommand(newResearchCmd())
	root.AddCommand(newWatchlistCmd())
	root.AddCommand(newScheduleCmd())
	root.AddCommand(newEmailCmd())
	return root
}

func newAPIClient() *client.Client {
	return client.NewWithOptions(client.Options{
		Profile: rootProfile,
		BaseURL: rootBaseURL,
		Token:   rootToken,
		Version: Version,
	})
}

func requireAPIClient() (*client.Client, error) {
	c := newAPIClient()
	if c.Token == "" {
		return nil, fmt.Errorf("missing API token; run `momentum auth save --profile dev --token TOKEN` or set MOMENTUMTICKER_TOKEN")
	}
	return c, nil
}

func printJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func stdinOrValue(cmd *cobra.Command, value string) (string, error) {
	if value != "" {
		return value, nil
	}
	stat, err := os.Stdin.Stat()
	if err == nil && stat.Mode()&os.ModeCharDevice == 0 {
		body, err := io.ReadAll(cmd.InOrStdin())
		return string(body), err
	}
	return "", nil
}

func readRequiredInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, "\n"), nil
	}
	body, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", err
	}
	text := string(body)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("missing entries; pass text arguments or pipe entries on stdin")
	}
	return text, nil
}

func printFolioTable(w io.Writer, title string, rows []folio) {
	filtered := make([]folio, 0, len(rows))
	for _, row := range rows {
		if row.ID != 0 || row.Name != "" || row.DisplayName != "" {
			filtered = append(filtered, row)
		}
	}
	if len(filtered) == 0 {
		return
	}
	fmt.Fprintln(w, "\n"+title)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tName\tKind\tAccess\tUpdated")
	for _, row := range filtered {
		name := row.DisplayName
		if name == "" {
			name = row.Name
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\n", row.ID, name, row.Kind, row.Access, row.HoldingsUpdatedAt)
	}
	_ = tw.Flush()
}

func printShareTable(w io.Writer, rows []share) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "no shares")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tFolio\tInviter\tRecipient\tPermission\tStatus")
	for _, row := range rows {
		name := row.Folio.DisplayName
		if name == "" {
			name = row.Folio.Name
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n", row.ID, name, row.InviterEmail, row.RecipientEmail, row.Permission, row.Status)
	}
	_ = tw.Flush()
}

func money(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func moneyPtr(value *float64) string {
	if value == nil {
		return "-"
	}
	return money(*value)
}

func percent(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}

func decimalPtr(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f", *value)
}
