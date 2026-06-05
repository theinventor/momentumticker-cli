package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "index",
		Short: "Read global index Folios such as S&P benchmarks",
	}
	c.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List global index Folios",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload struct {
				GlobalFolios       []folio `json:"global_folios"`
				LastPricingRefresh string  `json:"last_pricing_refresh"`
			}
			if err := api.DoJSON(http.MethodGet, "/api/v1/global_folios", nil, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			printFolioTable(cmd.OutOrStdout(), "Global indexes", payload.GlobalFolios)
			if payload.LastPricingRefresh != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "last pricing refresh: %s\n", payload.LastPricingRefresh)
			}
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use:     "show SYSTEM_KEY_OR_ID",
		Short:   "Show one global index Folio",
		Args:    cobra.ExactArgs(1),
		Example: "  momentum index show sp500",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload folio
			if err := api.DoJSON(http.MethodGet, "/api/v1/global_folios/"+url.PathEscape(args[0]), nil, nil, &payload); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), payload)
		},
	})
	return c
}

func newReportCmd() *cobra.Command {
	var folioID string
	c := &cobra.Command{
		Use:     "report",
		Short:   "Print the current momentum report",
		Example: "  momentum report\n  momentum report --folio master\n  momentum --json report --folio 42",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			q := url.Values{}
			if folioID != "" {
				q.Set("folio_id", folioID)
			}
			var payload report
			if err := api.DoJSON(http.MethodGet, "/api/v1/report", nil, q, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			if payload.Folio.Name != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Folio: %s (%s)\n", payload.Folio.DisplayName, payload.Folio.Access)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "Ticker\tPrice\tP/E\t5d\t1m\t3m\t6m\t1y\t3y\tShape")
			for _, row := range payload.Results {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					row.Ticker,
					money(row.Price),
					decimalPtr(row.TrailingPE),
					percent(row.Returns["five_days"]),
					percent(row.Returns["one_month"]),
					percent(row.Returns["three_months"]),
					percent(row.Returns["six_months"]),
					percent(row.Returns["one_year"]),
					percent(row.Returns["three_years"]),
					row.MomentumShape,
				)
			}
			return tw.Flush()
		},
	}
	c.Flags().StringVar(&folioID, "folio", "", "folio id, master, or global system key")
	return c
}

func newResearchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "research",
		Short: "Run and inspect ticker research scans",
	}
	c.AddCommand(newResearchRunCmd())
	c.AddCommand(newResearchListCmd())
	c.AddCommand(newResearchShowCmd())
	return c
}

func newResearchRunCmd() *cobra.Command {
	var name, tickers string
	c := &cobra.Command{
		Use:     "run",
		Short:   "Queue a research run from comma-separated tickers or stdin",
		Example: "  momentum research run --name Semis --tickers NVDA,AMD,AVGO,TSM\n  momentum research run --name Watch < tickers.txt",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			tickerText := strings.ReplaceAll(tickers, ",", "\n")
			if tickerText == "" {
				body, err := readRequiredInput(cmd, nil)
				if err != nil {
					return err
				}
				tickerText = body
			}
			var payload researchRun
			if err := api.DoJSON(http.MethodPost, "/api/v1/research_runs", map[string]string{"name": name, "tickers_text": tickerText}, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "research run queued: %d (%s)\n", payload.ID, payload.Name)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "CLI research run", "research run name")
	c.Flags().StringVar(&tickers, "tickers", "", "comma-separated tickers")
	return c
}

func newResearchListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List recent research runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload struct {
				ResearchRuns []researchRun `json:"research_runs"`
			}
			if err := api.DoJSON(http.MethodGet, "/api/v1/research_runs", nil, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tStatus\tTickers\tName")
			for _, run := range payload.ResearchRuns {
				fmt.Fprintf(tw, "%d\t%s\t%d\t%s\n", run.ID, run.Status, len(run.Tickers), run.Name)
			}
			return tw.Flush()
		},
	}
}

func newResearchShowCmd() *cobra.Command {
	var sort string
	c := &cobra.Command{
		Use:   "show RUN_ID",
		Short: "Show a research run and scored results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			q := url.Values{"sort": []string{sort}}
			var payload researchRun
			if err := api.DoJSON(http.MethodGet, "/api/v1/research_runs/"+url.PathEscape(args[0]), nil, q, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "Ticker\tPrice\tP/E\tShort\tLong\tOverall\t5d\t1m\t3m\t6m\t1y\t3y\tShape")
			for _, row := range payload.Results {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					row.Ticker,
					moneyPtr(row.Price),
					decimalPtr(row.TrailingPE),
					decimalPtr(row.ShortScore),
					decimalPtr(row.LongScore),
					decimalPtr(row.OverallScore),
					percent(row.Returns["five_days"]),
					percent(row.Returns["one_month"]),
					percent(row.Returns["three_months"]),
					percent(row.Returns["six_months"]),
					percent(row.Returns["one_year"]),
					percent(row.Returns["three_years"]),
					row.MomentumShape,
				)
			}
			return tw.Flush()
		},
	}
	c.Flags().StringVar(&sort, "sort", "overall", "sort key: overall, short, long, pe, or shape")
	return c
}

func newWatchlistCmd() *cobra.Command {
	c := &cobra.Command{
		Use:        "watchlist",
		Short:      "Compatibility alias for Master Folio holdings",
		Deprecated: "use `momentum folio holdings get master` and `momentum folio holdings set master`",
	}
	c.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Print Master Folio entries",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload struct {
				Entries string `json:"entries"`
			}
			if err := api.DoJSON(http.MethodGet, "/api/v1/watchlist", nil, nil, &payload); err != nil {
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
		Use:   "set [ENTRY...]",
		Short: "Replace Master Folio entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			entries, err := readRequiredInput(cmd, args)
			if err != nil {
				return err
			}
			var payload map[string]any
			if err := api.DoJSON(http.MethodPatch, "/api/v1/watchlist", map[string]string{"entries": entries}, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Master Folio updated")
			return nil
		},
	})
	return c
}

func newScheduleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "schedule",
		Short: "Get or update scheduled report delivery",
	}
	c.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Show report schedule settings",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			var payload map[string]any
			if err := api.DoJSON(http.MethodGet, "/api/v1/report_schedule", nil, nil, &payload); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), payload)
		},
	})
	c.AddCommand(newScheduleSetCmd())
	return c
}

func newScheduleSetCmd() *cobra.Command {
	var email, frequency, timezone string
	var enabled bool
	var hour int
	c := &cobra.Command{
		Use:     "set",
		Short:   "Update report schedule settings",
		Example: "  momentum schedule set --email kevin@example.com --enabled --frequency weekdays --hour 7 --timezone America/Los_Angeles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			body := map[string]any{
				"recipient_email": email,
				"enabled":         enabled,
				"frequency":       frequency,
				"delivery_hour":   hour,
				"timezone":        timezone,
			}
			var payload map[string]any
			if err := api.DoJSON(http.MethodPatch, "/api/v1/report_schedule", body, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "schedule updated")
			return nil
		},
	}
	c.Flags().StringVar(&email, "email", "", "recipient email")
	c.Flags().BoolVar(&enabled, "enabled", false, "enable scheduled reports")
	c.Flags().StringVar(&frequency, "frequency", "daily", "daily, weekdays, or weekly")
	c.Flags().IntVar(&hour, "hour", 7, "delivery hour, 0-23")
	c.Flags().StringVar(&timezone, "timezone", "America/Los_Angeles", "IANA time zone")
	return c
}

func newEmailCmd() *cobra.Command {
	var recipient string
	c := &cobra.Command{
		Use:     "email [RECIPIENT]",
		Short:   "Queue an immediate report email",
		Example: "  momentum email kevin@example.com\n  momentum email --recipient kevin@example.com",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := requireAPIClient()
			if err != nil {
				return err
			}
			if recipient == "" && len(args) > 0 {
				recipient = args[0]
			}
			body := map[string]string{}
			if recipient != "" {
				body["recipient_email"] = recipient
			}
			var payload map[string]any
			if err := api.DoJSON(http.MethodPost, "/api/v1/report_emails", body, nil, &payload); err != nil {
				return err
			}
			if rootJSON {
				return printJSON(cmd.OutOrStdout(), payload)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "report email queued")
			return nil
		},
	}
	c.Flags().StringVar(&recipient, "recipient", "", "recipient email")
	return c
}
