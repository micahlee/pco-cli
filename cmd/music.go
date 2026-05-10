package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var musicCmd = &cobra.Command{
	Use:   "music",
	Short: "Music team management",
}

var musicTeamCmd = &cobra.Command{
	Use:   "team",
	Short: "List Band team members with IDs and typical positions",
	RunE: func(cmd *cobra.Command, args []string) error {
		members, err := svc.ListBandMembers(cmd.Context())
		if err != nil {
			return err
		}

		headers := []string{"Person ID", "Name", "Typical Positions"}
		rows := make([][]string, len(members))
		for i, m := range members {
			rows[i] = []string{m.PersonID, m.Name, m.TypicalPositions}
		}
		printer.Table(headers, rows)

		fmt.Fprintf(printer.Writer(), "\nBand team ID: %s  |  Service Responsibilities team ID: %s\n",
			svc.Config.BandTeamID, svc.Config.ServiceRespTeamID)
		fmt.Fprintln(printer.Writer(), "Music Lead position is in Service Responsibilities team.")
		return nil
	},
}

var musicAvailabilityCmd = &cobra.Command{
	Use:   "availability <YYYY-MM-DD>",
	Short: "Check who's available on a date",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		results, err := svc.CheckAvailability(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		var available, blocked []string

		for _, r := range results {
			if r.Available {
				available = append(available, fmt.Sprintf("  %-12s %-25s %s",
					r.Member.PersonID, r.Member.Name, r.Member.TypicalPositions))
			} else {
				blocked = append(blocked, fmt.Sprintf("  %-12s %-25s %s",
					r.Member.PersonID, r.Member.Name, r.Reason))
			}
		}

		w := printer.Writer()
		fmt.Fprintf(w, "Band team availability for %s\n\n", args[0])
		fmt.Fprintf(w, "Available (%d):\n", len(available))
		for _, line := range available {
			fmt.Fprintln(w, line)
		}

		if len(blocked) > 0 {
			fmt.Fprintf(w, "\nBlocked out (%d):\n", len(blocked))
			for _, line := range blocked {
				fmt.Fprintln(w, line)
			}
		}
		return nil
	},
}

var musicMonthCmd = &cobra.Command{
	Use:   "month <YYYY-MM>",
	Short: "Full music scheduling overview for a month",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := svc.MusicMonth(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if len(result.Plans) == 0 {
			fmt.Fprintf(printer.Writer(), "No plans found for %s.\n", args[0])
			return nil
		}

		w := printer.Writer()
		fmt.Fprintf(w, "Music schedule overview -- %s\n\n", result.YearMonth)
		fmt.Fprintln(w, strings.Repeat("=", 70))

		for _, plan := range result.Plans {
			fmt.Fprintf(w, "\n%s  [%s]  %s\n", plan.Date, plan.PlanID, plan.Title)
			fmt.Fprintln(w, strings.Repeat("-", 60))

			// Music Lead
			if plan.MusicLead != nil {
				flag := ""
				if plan.MusicLead.Attrs.Status != "C" {
					flag = fmt.Sprintf(" ⚠ %s", plan.MusicLead.Attrs.Status)
				}
				fmt.Fprintf(w, "  ML:   %s%s\n", plan.MusicLead.Attrs.Name, flag)
			} else {
				fmt.Fprintln(w, "  ML:   *** NONE — must assign before scheduling ***")
			}

			// Band
			if len(plan.BandMembers) > 0 {
				for _, m := range plan.BandMembers {
					flag := ""
					if m.Attrs.Status != "C" {
						flag = fmt.Sprintf(" (%s)", m.Attrs.Status)
					}
					fmt.Fprintf(w, "  Band: %-20s %s%s\n", m.Attrs.TeamPositionName, m.Attrs.Name, flag)
				}
			} else {
				fmt.Fprintln(w, "  Band: (none scheduled)")
			}

			// Blocked
			if len(plan.BlockedNames) > 0 {
				fmt.Fprintf(w, "  Out:  %s\n", strings.Join(plan.BlockedNames, ", "))
			}
		}

		// Appearance counts
		fmt.Fprintf(w, "\n%s\n", strings.Repeat("=", 70))
		fmt.Fprintln(w, "Appearance counts this month:")

		type nameCount struct {
			Name  string
			Count int
		}
		var counts []nameCount
		for name, count := range result.AppearanceCounts {
			if count > 0 {
				counts = append(counts, nameCount{name, count})
			}
		}
		sort.Slice(counts, func(i, j int) bool {
			return counts[i].Name < counts[j].Name
		})

		for _, nc := range counts {
			bar := strings.Repeat("*", nc.Count)
			over := ""
			if nc.Count > 2 {
				over = " (over limit)"
			}
			fmt.Fprintf(w, "  %-25s %s %d%s\n", nc.Name, bar, nc.Count, over)
		}

		return nil
	},
}

func init() {
	musicCmd.AddCommand(musicTeamCmd)
	musicCmd.AddCommand(musicAvailabilityCmd)
	musicCmd.AddCommand(musicMonthCmd)
	rootCmd.AddCommand(musicCmd)
}
