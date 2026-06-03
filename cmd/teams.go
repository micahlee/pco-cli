package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/micahlee/pco-cli/internal/models"
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
		result, err := svc.EnableSignups(cmd.Context(), args[0], teamID)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(result)
		}
		printEnableSignupResult(result)
		return nil
	},
}

var teamsSignupsCmd = &cobra.Command{
	Use:   "signups <plan-id>",
	Short: "List team sign-ups for a plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teamID, _ := cmd.Flags().GetString("team-id")
		signups, err := svc.ListTeamSignups(cmd.Context(), args[0], teamID)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(signups)
		}
		printTeamSignupTable(signups)
		return nil
	},
}

var teamsSignupsMonthCmd = &cobra.Command{
	Use:   "signups-month <YYYY-MM>",
	Short: "List team sign-ups for every plan in a month",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teamID, _ := cmd.Flags().GetString("team-id")
		signups, err := svc.ListTeamSignupsMonth(cmd.Context(), args[0], teamID)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(signups)
		}
		printTeamSignupTable(signups)
		return nil
	},
}

var teamsEnableSignupsMonthCmd = &cobra.Command{
	Use:   "enable-signups-month <YYYY-MM>",
	Short: "Enable team sign-ups for every plan in a month",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teamID, _ := cmd.Flags().GetString("team-id")
		results, err := svc.EnableSignupsMonth(cmd.Context(), args[0], teamID)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(results)
		}

		headers := []string{"Plan ID", "TeamSignup ID", "Team", "Status", "Result"}
		rows := make([][]string, len(results))
		for i, result := range results {
			rows[i] = []string{
				result.PlanID,
				result.TeamSignup.ID,
				teamSignupName(result.TeamSignup),
				signupStatus(result.TeamSignup),
				enableSignupAction(result),
			}
		}
		printer.Table(headers, rows)
		return nil
	},
}

func init() {
	teamsEnableSignupsCmd.Flags().String("team-id", "", "team ID (defaults to Band team)")
	teamsSignupsCmd.Flags().String("team-id", "", "team ID (defaults to Band team)")
	teamsSignupsMonthCmd.Flags().String("team-id", "", "team ID (defaults to Band team)")
	teamsEnableSignupsMonthCmd.Flags().String("team-id", "", "team ID (defaults to Band team)")

	teamsCmd.AddCommand(teamsShowCmd)
	teamsCmd.AddCommand(teamsScheduleCmd)
	teamsCmd.AddCommand(teamsUnscheduleCmd)
	teamsCmd.AddCommand(teamsEnableSignupsCmd)
	teamsCmd.AddCommand(teamsSignupsCmd)
	teamsCmd.AddCommand(teamsSignupsMonthCmd)
	teamsCmd.AddCommand(teamsEnableSignupsMonthCmd)
	rootCmd.AddCommand(teamsCmd)
}

func printTeamSignupTable(signups []models.TeamSignup) {
	headers := []string{"Plan ID", "TeamSignup ID", "Team", "Status", "Attributes"}
	rows := make([][]string, len(signups))
	for i, signup := range signups {
		rows[i] = []string{
			signup.PlanID,
			signup.ID,
			teamSignupName(signup),
			signupStatus(signup),
			teamSignupAttrsSummary(signup),
		}
	}
	printer.Table(headers, rows)
}

func printEnableSignupResult(result *models.TeamSignupEnableResult) {
	teamName := teamSignupName(result.TeamSignup)
	action := enableSignupAction(*result)
	if action == "already enabled" {
		fmt.Fprintf(printer.Writer(), "Sign-ups already enabled for %s on plan %s (TeamSignup ID: %s)\n",
			teamName, result.PlanID, result.TeamSignup.ID)
		return
	}
	fmt.Fprintf(printer.Writer(), "Sign-ups enabled for %s on plan %s (TeamSignup ID: %s)\n",
		teamName, result.PlanID, result.TeamSignup.ID)
}

func teamSignupName(signup models.TeamSignup) string {
	if signup.TeamName != "" {
		return fmt.Sprintf("%s (%s)", signup.TeamName, signup.TeamID)
	}
	if signup.TeamID != "" {
		if signup.TeamID == svc.Config.BandTeamID {
			return fmt.Sprintf("Band (%s)", signup.TeamID)
		}
		return "team " + signup.TeamID
	}
	return ""
}

func signupStatus(signup models.TeamSignup) string {
	if signup.Attrs.SignupsEnabled == nil {
		return "unknown"
	}
	if *signup.Attrs.SignupsEnabled {
		return "open"
	}
	return "closed"
}

func enableSignupAction(result models.TeamSignupEnableResult) string {
	if result.Created {
		return "created"
	}
	if result.Updated {
		return "updated"
	}
	return "already enabled"
}

func teamSignupAttrsSummary(signup models.TeamSignup) string {
	attrs := make(map[string]any)
	for key, value := range signup.Attrs.Raw {
		if key == "signups_enabled" {
			continue
		}
		attrs[key] = value
	}
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		data, err := json.Marshal(attrs[key])
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, string(data)))
	}
	return strings.Join(parts, ", ")
}
