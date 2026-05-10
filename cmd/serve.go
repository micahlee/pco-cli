package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Manage serve requests",
}

var serveRequestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "List pending serve requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		requests, err := svc.ListServeRequests(cmd.Context())
		if err != nil {
			return err
		}

		if len(requests) == 0 {
			fmt.Fprintln(printer.Writer(), "No pending serve requests.")
			return nil
		}

		headers := []string{"ID", "Date", "Position", "Status"}
		rows := make([][]string, len(requests))
		for i, r := range requests {
			date := r.Attrs.SortDate
			if len(date) >= 10 {
				date = date[:10]
			}
			rows[i] = []string{r.ID, date, r.Attrs.TeamPositionName, r.Attrs.Status}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var serveAcceptCmd = &cobra.Command{
	Use:   "accept <schedule-id>",
	Short: "Accept a serve request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := svc.AcceptServeRequest(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Fprintf(printer.Writer(), "Accepted serve request %s\n", args[0])
		return nil
	},
}

var serveDeclineCmd = &cobra.Command{
	Use:   "decline <schedule-id>",
	Short: "Decline a serve request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := svc.DeclineServeRequest(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Fprintf(printer.Writer(), "Declined serve request %s\n", args[0])
		return nil
	},
}

func init() {
	serveCmd.AddCommand(serveRequestsCmd)
	serveCmd.AddCommand(serveAcceptCmd)
	serveCmd.AddCommand(serveDeclineCmd)
	rootCmd.AddCommand(serveCmd)
}
