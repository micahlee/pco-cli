package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blockoutsCmd = &cobra.Command{
	Use:   "blockouts",
	Short: "Manage blockout dates",
}

var blockoutsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List my blockout dates",
	RunE: func(cmd *cobra.Command, args []string) error {
		blockouts, err := svc.ListBlockouts(cmd.Context())
		if err != nil {
			return err
		}

		if len(blockouts) == 0 {
			fmt.Fprintln(printer.Writer(), "No blockout dates.")
			return nil
		}

		headers := []string{"ID", "Start", "End", "Reason"}
		rows := make([][]string, len(blockouts))
		for i, b := range blockouts {
			start := b.Attrs.StartsAt
			if len(start) >= 10 {
				start = start[:10]
			}
			end := b.Attrs.EndsAt
			if len(end) >= 10 {
				end = end[:10]
			}
			rows[i] = []string{b.ID, start, end, b.Attrs.Reason}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var blockoutsAddCmd = &cobra.Command{
	Use:   "add <start-date> <end-date>",
	Short: "Add a blockout date (YYYY-MM-DD)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		reason, _ := cmd.Flags().GetString("reason")
		blockout, err := svc.AddBlockout(cmd.Context(), args[0], args[1], reason)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("Created blockout %s: %s to %s", blockout.ID, args[0], args[1])
		if reason != "" {
			msg += fmt.Sprintf(" (%s)", reason)
		}
		fmt.Fprintln(printer.Writer(), msg)
		return nil
	},
}

var blockoutsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a blockout date",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := svc.DeleteBlockout(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Fprintf(printer.Writer(), "Deleted blockout %s\n", args[0])
		return nil
	},
}

func init() {
	blockoutsAddCmd.Flags().String("reason", "", "reason for the blockout")

	blockoutsCmd.AddCommand(blockoutsListCmd)
	blockoutsCmd.AddCommand(blockoutsAddCmd)
	blockoutsCmd.AddCommand(blockoutsDeleteCmd)
	rootCmd.AddCommand(blockoutsCmd)
}
