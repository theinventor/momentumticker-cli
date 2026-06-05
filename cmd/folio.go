package cmd

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

func newFolioCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "folio",
		Short: "Manage owned, shared, and global Folios",
		Long: `Manage normal-user Folios through the public MomentumTicker API.

Owned Folios can be created, renamed, edited, and shared. Shared Folios can be
read or edited only when the server grants the current token permission.`,
	}
	c.AddCommand(newFolioListCmd())
	c.AddCommand(newFolioShowCmd())
	c.AddCommand(newFolioCreateCmd())
	c.AddCommand(newFolioUpdateCmd())
	c.AddCommand(newFolioHoldingsCmd())
	c.AddCommand(newFolioShareCmd())
	return c
}

func newFolioListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List owned Folios, shared Folios, invitations, and global indexes",
		Example: "  momentum folio list\n  momentum --json folio list",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload struct {
				MasterFolio        folio   `json:"master_folio"`
				ResearchFolios     []folio `json:"research_folios"`
				SharedFolios       []folio `json:"shared_folios"`
				PendingInvitations []share `json:"pending_invitations"`
				GlobalFolios       []folio `json:"global_folios"`
			}
			if err := c.DoJSON(http.MethodGet, "/api/v1/folios", nil, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			printFolioTable(cmd.OutOrStdout(), "Owned", append([]folio{payload.MasterFolio}, payload.ResearchFolios...))
			printFolioTable(cmd.OutOrStdout(), "Shared with me", payload.SharedFolios)
			printFolioTable(cmd.OutOrStdout(), "Global indexes", payload.GlobalFolios)
			if len(payload.PendingInvitations) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nPending invitations")
				printShareTable(cmd.OutOrStdout(), payload.PendingInvitations)
			}
			return nil
		},
	}
}

func newFolioShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show FOLIO_ID",
		Short:   "Show one accessible Folio",
		Args:    cobra.ExactArgs(1),
		Example: "  momentum folio show 42\n  momentum folio show master",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload folio
			if err := c.DoJSON(http.MethodGet, "/api/v1/folios/"+url.PathEscape(args[0]), nil, nil, &payload); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), payload)
		},
	}
}

func newFolioCreateCmd() *cobra.Command {
	var name, description, entries string
	c := &cobra.Command{
		Use:     "create",
		Short:   "Create a Research Folio",
		Example: "  momentum folio create --name \"AI Research\" --entries \"NVDA 5\\nAMD 10\"\n  momentum folio create --name \"Semis\" < entries.txt",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			body := map[string]string{"name": name, "description": description}
			text, err := stdinOrValue(cmd, entries)
			if err != nil {
				return err
			}
			if text != "" {
				body["entries"] = text
			}
			var payload folio
			if err := api.DoJSON(http.MethodPost, "/api/v1/folios", body, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created folio %d: %s\n", payload.ID, payload.Name)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "Research Folio", "folio name")
	c.Flags().StringVar(&description, "description", "", "folio description")
	c.Flags().StringVar(&entries, "entries", "", "ticker entries; reads stdin when blank and stdin is piped")
	return c
}

func newFolioUpdateCmd() *cobra.Command {
	var name, description, entries string
	c := &cobra.Command{
		Use:     "update FOLIO_ID",
		Short:   "Update Folio metadata or entries",
		Args:    cobra.ExactArgs(1),
		Example: "  momentum folio update 42 --name \"Updated Folio\"\n  momentum folio update 42 --entries \"AAPL 2\\nMSFT 3\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			body := map[string]string{}
			if name != "" {
				body["name"] = name
			}
			if description != "" {
				body["description"] = description
			}
			text, err := stdinOrValue(cmd, entries)
			if err != nil {
				return err
			}
			if text != "" {
				body["entries"] = text
			}
			var payload folio
			if err := api.DoJSON(http.MethodPatch, "/api/v1/folios/"+url.PathEscape(args[0]), body, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated folio %d: %s\n", payload.ID, payload.Name)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "folio name")
	c.Flags().StringVar(&description, "description", "", "folio description")
	c.Flags().StringVar(&entries, "entries", "", "ticker entries; reads stdin when blank and stdin is piped")
	return c
}

func newFolioHoldingsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "holdings",
		Short: "Get or replace Folio holdings entries",
	}
	c.AddCommand(&cobra.Command{
		Use:     "get FOLIO_ID",
		Short:   "Print Folio holdings as editable text",
		Args:    cobra.ExactArgs(1),
		Example: "  momentum folio holdings get 42 > entries.txt",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload struct {
				Folio    folio     `json:"folio"`
				Entries  string    `json:"entries"`
				Holdings []holding `json:"holdings"`
			}
			if err := api.DoJSON(http.MethodGet, "/api/v1/folios/"+url.PathEscape(args[0])+"/holdings", nil, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprint(cmd.OutOrStdout(), payload.Entries)
			if payload.Entries == "" || payload.Entries[len(payload.Entries)-1:] != "\n" {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use:     "set FOLIO_ID [ENTRY...]",
		Short:   "Replace Folio holdings from arguments or stdin",
		Args:    cobra.MinimumNArgs(1),
		Example: "  momentum folio holdings set 42 AAPL 2 MSFT 3\n  momentum folio holdings set 42 < entries.txt",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			entries, err := readRequiredInput(cmd, args[1:])
			if err != nil {
				return err
			}
			var payload map[string]any
			if err := api.DoJSON(http.MethodPatch, "/api/v1/folios/"+url.PathEscape(args[0])+"/holdings", map[string]string{"entries": entries}, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "folio holdings updated")
			return nil
		},
	})
	return c
}

func newFolioShareCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "share",
		Short: "Manage Folio share invitations and permissions",
	}
	c.AddCommand(newShareListCmd())
	c.AddCommand(newShareInviteCmd())
	c.AddCommand(newShareStatusCmd("accept", "accepted"))
	c.AddCommand(newShareStatusCmd("decline", "declined"))
	c.AddCommand(newShareRevokeCmd())
	return c
}

func newShareListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List received, accepted, and sent Folio shares",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload struct {
				Received []share `json:"received"`
				Accepted []share `json:"accepted"`
				Sent     []share `json:"sent"`
			}
			if err := api.DoJSON(http.MethodGet, "/api/v1/folio_shares", nil, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			printShareTable(cmd.OutOrStdout(), append(append(payload.Received, payload.Accepted...), payload.Sent...))
			return nil
		},
	}
}

func newShareInviteCmd() *cobra.Command {
	var email, permission string
	c := &cobra.Command{
		Use:     "invite FOLIO_ID --email EMAIL [--permission read|write]",
		Short:   "Invite another user to read or write a Folio",
		Args:    cobra.ExactArgs(1),
		Example: "  momentum folio share invite 42 --email friend@example.com --permission write",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload share
			body := map[string]string{"recipient_email": email, "permission": permission}
			if err := api.DoJSON(http.MethodPost, "/api/v1/folios/"+url.PathEscape(args[0])+"/shares", body, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "share %d pending for %s\n", payload.ID, payload.RecipientEmail)
			return nil
		},
	}
	c.Flags().StringVar(&email, "email", "", "recipient email")
	c.Flags().StringVar(&permission, "permission", "read", "permission: read or write")
	return c
}

func newShareStatusCmd(verb, status string) *cobra.Command {
	return &cobra.Command{
		Use:   verb + " SHARE_ID",
		Short: verb + " a pending Folio share",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload share
			if err := api.DoJSON(http.MethodPatch, "/api/v1/folio_shares/"+url.PathEscape(args[0]), map[string]string{"status": status}, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "share %d %s\n", payload.ID, payload.Status)
			return nil
		},
	}
}

func newShareRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke SHARE_ID",
		Short: "Revoke a sent Folio share",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload share
			if err := api.DoJSON(http.MethodDelete, "/api/v1/folio_shares/"+url.PathEscape(args[0]), nil, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "share %d revoked\n", payload.ID)
			return nil
		},
	}
}
