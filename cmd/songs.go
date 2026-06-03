package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/micahlee/pco-cli/internal/pco"
	"github.com/spf13/cobra"
)

var songsCmd = &cobra.Command{
	Use:   "songs",
	Short: "Manage songs",
}

var songsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search songs in the library",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		includeArrangements, _ := cmd.Flags().GetBool("include-arrangements")
		if includeArrangements {
			songs, err := svc.SearchSongsWithArrangements(cmd.Context(), query)
			if err != nil {
				return err
			}
			if jsonOutput {
				return printer.JSON(songs)
			}

			headers := []string{"ID", "Title", "Author", "Arrangements"}
			rows := make([][]string, len(songs))
			for i, s := range songs {
				rows[i] = []string{s.ID, s.Title, s.Author, strconv.Itoa(len(s.Arrangements))}
			}
			printer.Table(headers, rows)
			return nil
		}

		songs, err := svc.SearchSongs(cmd.Context(), query)
		if err != nil {
			return err
		}

		headers := []string{"ID", "Title", "Author"}
		rows := make([][]string, len(songs))
		for i, s := range songs {
			rows[i] = []string{s.ID, s.Attrs.Title, s.Attrs.Author}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var songsHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show song usage history",
	Long:  "Shows each active song's last-used date, total use count, and all plan dates. Use to avoid recently overused songs when selecting for a series.",
	RunE: func(cmd *cobra.Command, args []string) error {
		weeks, _ := cmd.Flags().GetInt("weeks")
		usage, planCount, err := svc.SongHistory(cmd.Context(), weeks)
		if err != nil {
			return err
		}

		if jsonOutput {
			return printer.JSON(struct {
				Weeks     int `json:"weeks"`
				PlanCount int `json:"plan_count"`
				Songs     any `json:"songs"`
			}{
				Weeks:     weeks,
				PlanCount: planCount,
				Songs:     usage,
			})
		}

		fmt.Fprintf(printer.Writer(), "Song history — past %d weeks (active songs only, %d plans)\n\n", weeks, planCount)

		headers := []string{"Song", "Uses", "Last Used", "All Dates"}
		rows := make([][]string, len(usage))
		for i, u := range usage {
			rows[i] = []string{
				u.Title,
				strconv.Itoa(u.Uses),
				u.LastUsed,
				strings.Join(u.Dates, ", "),
			}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var songsArrangementsCmd = &cobra.Command{
	Use:   "arrangements <song-id>",
	Short: "List arrangements for a song with tempo and meter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arrangements, err := svc.ListSongArrangements(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if jsonOutput {
			return printer.JSON(arrangements)
		}

		headers := []string{"ID", "Name", "BPM", "Meter", "Length", "Key", "Archived", "Updated"}
		rows := make([][]string, len(arrangements))
		for i, a := range arrangements {
			rows[i] = []string{
				a.ID,
				a.Attrs.Name,
				formatFloat(a.Attrs.BPM),
				a.Attrs.Meter,
				formatInt(a.Attrs.Length),
				a.Attrs.ChordChartKey,
				strconv.FormatBool(a.Archived),
				a.Attrs.UpdatedAt,
			}
		}
		printer.Table(headers, rows)
		return nil
	},
}

var songsSetCmd = &cobra.Command{
	Use:   "set <plan-id> <item-id> <song-id>",
	Short: "Assign a song to an existing plan item",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := songAssignmentOptions(cmd)
		if err != nil {
			return err
		}
		result, err := svc.SetSong(cmd.Context(), args[0], args[1], args[2], opts)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(result)
		}
		fmt.Fprintf(printer.Writer(), "Set item %s to %s", args[1], result.Title)
		printAssignmentArrangement(result)
		fmt.Fprintf(printer.Writer(), " (plan %s)\n", args[0])
		return nil
	},
}

var songsAddCmd = &cobra.Command{
	Use:   "add <plan-id> <after-item-id> <song-id>",
	Short: "Insert a new song item after the specified item",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		opts, err := songAssignmentOptions(cmd)
		if err != nil {
			return err
		}
		result, err := svc.AddSongItem(cmd.Context(), args[0], args[1], args[2], label, opts)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(result)
		}
		fmt.Fprintf(printer.Writer(), "Added %s after item %s (new item ID: %s", result.Title, args[1], result.ItemID)
		printAssignmentArrangement(result)
		fmt.Fprintln(printer.Writer(), ")")
		return nil
	},
}

func init() {
	songsSearchCmd.Flags().String("query", "", "search by song title")
	songsSearchCmd.Flags().Bool("include-arrangements", false, "include arrangement summaries in search results")
	songsHistoryCmd.Flags().Int("weeks", 16, "number of weeks to look back")
	songsAddCmd.Flags().String("label", "", "label to append in parentheses (e.g. \"Lord's Supper\")")
	songsSetCmd.Flags().String("arrangement-id", "", "arrangement ID to attach")
	songsSetCmd.Flags().Bool("song-only", false, "leave arrangement blank")
	songsAddCmd.Flags().String("arrangement-id", "", "arrangement ID to attach")
	songsAddCmd.Flags().Bool("song-only", false, "leave arrangement blank")

	songsCmd.AddCommand(songsSearchCmd)
	songsCmd.AddCommand(songsHistoryCmd)
	songsCmd.AddCommand(songsArrangementsCmd)
	songsCmd.AddCommand(songsSetCmd)
	songsCmd.AddCommand(songsAddCmd)
	rootCmd.AddCommand(songsCmd)
}

func formatFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func formatInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func songAssignmentOptions(cmd *cobra.Command) (pco.SongAssignmentOptions, error) {
	arrangementID, _ := cmd.Flags().GetString("arrangement-id")
	songOnly, _ := cmd.Flags().GetBool("song-only")
	if arrangementID != "" && songOnly {
		return pco.SongAssignmentOptions{}, fmt.Errorf("--arrangement-id and --song-only cannot be used together")
	}
	return pco.SongAssignmentOptions{ArrangementID: arrangementID, SongOnly: songOnly}, nil
}

func printAssignmentArrangement(result *pco.SongAssignmentResult) {
	if result.ArrangementID != "" {
		name := result.ArrangementName
		if name == "" {
			name = "arrangement"
		}
		fmt.Fprintf(printer.Writer(), ", arrangement %s (%s)", name, result.ArrangementID)
		return
	}
	if result.Warning != "" {
		fmt.Fprintf(printer.Writer(), " [warning: %s]", result.Warning)
	}
}
