package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var teamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "Manage team assignments",
}

var teamsShowCmd = &cobra.Command{
	Use:   "show <plan-id>",
	Short: "List team members for a plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		members, err := svc.ListTeamMembers(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		headers := []string{"Assign ID", "Name", "Team Position", "Status"}
		rows := make([][]string, len(members))
		for i, m := range members {
			rows[i] = []string{m.ID, m.Attrs.Name, m.Attrs.TeamPositionName, m.Attrs.Status}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var teamsScheduleCmd = &cobra.Command{
	Use:   "schedule <plan-id> <person-id> <team-id> <position>",
	Short: "Assign a person to a plan position (notification queued)",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := svc.SchedulePerson(cmd.Context(), args[0], args[1], args[2], args[3])
		if err != nil {
			return err
		}
		fmt.Fprintf(printer.Writer(), "Scheduled %s as %s (notification queued, not sent)\n", name, args[3])
		return nil
	},
}

var teamsUnscheduleCmd = &cobra.Command{
	Use:   "unschedule <plan-id> <assign-id>",
	Short: "Remove an assignment from a plan",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := svc.UnschedulePerson(cmd.Context(), args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(printer.Writer(), "Removed assignment %s from plan %s\n", args[1], args[0])
		return nil
	},
}

var teamsEnableSignupsCmd = &cobra.Command{
	Use:   "enable-signups <plan-id>",
	Short: "Enable team sign-ups for a plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teamID, _ := cmd.Flags().GetString("team-id")
		signupID, err := svc.EnableSignups(cmd.Context(), args[0], teamID)
		if err != nil {
			return err
		}
		teamName := "Band"
		if teamID != "" && teamID != svc.Config.BandTeamID {
			teamName = "team " + teamID
		}
		fmt.Fprintf(printer.Writer(), "Sign-ups enabled for %s on plan %s (TeamSignup ID: %s)\n",
			teamName, args[0], signupID)
		return nil
	},
}

func init() {
	teamsEnableSignupsCmd.Flags().String("team-id", "", "team ID (defaults to Band team)")

	teamsCmd.AddCommand(teamsShowCmd)
	teamsCmd.AddCommand(teamsScheduleCmd)
	teamsCmd.AddCommand(teamsUnscheduleCmd)
	teamsCmd.AddCommand(teamsEnableSignupsCmd)
	rootCmd.AddCommand(teamsCmd)
}
