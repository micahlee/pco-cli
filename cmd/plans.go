package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var plansCmd = &cobra.Command{
	Use:   "plans",
	Short: "Manage service plans",
}

var plansListCmd = &cobra.Command{
	Use:   "list",
	Short: "List upcoming plans",
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")
		plans, err := svc.ListPlans(cmd.Context(), "future", count)
		if err != nil {
			return err
		}

		headers := []string{"ID", "Date", "Title"}
		rows := make([][]string, len(plans))
		for i, p := range plans {
			title := p.Attrs.Title
			if title == "" {
				title = "(no title)"
			}
			rows[i] = []string{p.ID, p.Attrs.Dates, title}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var plansShowCmd = &cobra.Command{
	Use:   "show <plan-id>",
	Short: "Show plan details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]

		plan, err := svc.GetPlan(cmd.Context(), planID)
		if err != nil {
			return err
		}

		// Plan header
		title := plan.Attrs.Title
		if title == "" {
			title = "(no title)"
		}
		fmt.Fprintf(printer.Writer(), "Plan: %s\nDate: %s\n\n", title, plan.Attrs.Dates)

		// Items
		items, err := svc.ListPlanItems(cmd.Context(), planID)
		if err != nil {
			return err
		}

		fmt.Fprintln(printer.Writer(), "Items:")
		for _, item := range items {
			length := item.Attrs.Length
			timeStr := ""
			if length > 0 {
				timeStr = fmt.Sprintf(" %d:%02d", length/60, length%60)
			}
			title := item.Attrs.Title
			if title == "" {
				title = item.Attrs.Description
			}
			fmt.Fprintf(printer.Writer(), "  [%-10s] %s%s\n", item.Attrs.ItemType, title, timeStr)
		}

		// Songs
		songs, err := svc.ListPlanSongs(cmd.Context(), planID)
		if err != nil {
			return err
		}
		if len(songs) > 0 {
			fmt.Fprintln(printer.Writer(), "\nSongs:")
			for _, s := range songs {
				fmt.Fprintf(printer.Writer(), "  - %s (key: %s)\n", s.Attrs.Title, s.Attrs.KeyName)
			}
		}

		return nil
	},
}

var plansItemsCmd = &cobra.Command{
	Use:   "items <plan-id>",
	Short: "List plan items with IDs, sequences, and song assignments",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := svc.ListPlanItems(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		headers := []string{"Item ID", "Seq", "Type", "Title", "Song ID"}
		rows := make([][]string, len(items))
		for i, item := range items {
			rows[i] = []string{
				item.ID,
				strconv.Itoa(item.Attrs.Sequence),
				item.Attrs.ItemType,
				item.Attrs.Title,
				item.SongID,
			}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var plansTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List plan templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		templates, err := svc.ListTemplates(cmd.Context())
		if err != nil {
			return err
		}

		headers := []string{"ID", "Name"}
		rows := make([][]string, len(templates))
		for i, t := range templates {
			rows[i] = []string{t.ID, t.Attrs.Name}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var plansCreateCmd = &cobra.Command{
	Use:   "create <YYYY-MM-DD>",
	Short: "Create a plan from a template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateID, _ := cmd.Flags().GetString("template")
		plans, err := svc.CreatePlan(cmd.Context(), args[0], templateID)
		if err != nil {
			return err
		}

		for _, p := range plans {
			fmt.Fprintf(printer.Writer(), "Created plan %s: %s — %s\n",
				p.ID, p.Attrs.Dates, p.Attrs.PlanningCenterURL)
		}
		return nil
	},
}

func init() {
	plansListCmd.Flags().Int("count", 5, "number of plans to show")
	plansCreateCmd.Flags().String("template", "", "template ID (defaults to Sunday Morning Worship)")

	plansCmd.AddCommand(plansListCmd)
	plansCmd.AddCommand(plansShowCmd)
	plansCmd.AddCommand(plansItemsCmd)
	plansCmd.AddCommand(plansTemplatesCmd)
	plansCmd.AddCommand(plansCreateCmd)
	rootCmd.AddCommand(plansCmd)
}
