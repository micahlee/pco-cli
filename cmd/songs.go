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

		if jsonOutput {
			return printer.JSON(songs)
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

var songsReplaceCmd = &cobra.Command{
	Use:   "replace",
	Short: "Safely replace one song in a plan by date and title",
	Long:  "Resolve one plan, current item, replacement library song, and arrangement; then recheck the current item immediately before updating and verify it afterward.",
	Example: `  pco songs replace --date 2026-09-13 --current "Ancient of Days" --with "I Am Not My Own"
  pco songs replace --plan-id 123 --current "Ancient of Days" --with "I Am Not My Own" --dry-run --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		date, _ := cmd.Flags().GetString("date")
		planID, _ := cmd.Flags().GetString("plan-id")
		current, _ := cmd.Flags().GetString("current")
		replacement, _ := cmd.Flags().GetString("with")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		arrangementID, _ := cmd.Flags().GetString("arrangement-id")
		songOnly, _ := cmd.Flags().GetBool("song-only")
		result, err := svc.ReplaceSong(cmd.Context(), pco.ReplaceSongOptions{
			Date: date, PlanID: planID, CurrentTitle: current, Replacement: replacement,
			ArrangementID: arrangementID, SongOnly: songOnly, DryRun: dryRun,
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(result)
		}
		switch result.Status {
		case "preview":
			fmt.Fprintf(printer.Writer(), "Preview: replace %s with %s in plan %s (%s), item %s", result.Current.Title, result.Replacement.Title, result.Plan.ID, result.Plan.Date, result.Item.ID)
		case "noop":
			fmt.Fprintf(printer.Writer(), "No change: %s is already present in plan %s (%s), item %s", result.Replacement.Title, result.Plan.ID, result.Plan.Date, result.Item.ID)
		default:
			fmt.Fprintf(printer.Writer(), "Replaced %s with %s in plan %s (%s), item %s; verified", result.Current.Title, result.Replacement.Title, result.Plan.ID, result.Plan.Date, result.Item.ID)
		}
		if result.Arrangement != nil {
			fmt.Fprintf(printer.Writer(), ", arrangement %s (%s)", result.Arrangement.Name, result.Arrangement.ID)
		}
		fmt.Fprintln(printer.Writer())
		for _, warning := range result.Warnings {
			fmt.Fprintf(printer.Writer(), "Warning: %s\n", warning)
		}
		return nil
	},
}

var songsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Explicitly create a library song and its initial arrangement",
	Long:  "Create a song library record and its initial arrangement. This command never places the song in a service plan. Duplicate matches stop creation unless --allow-duplicate is explicit.",
	Example: `  pco songs create --title "Example Song" --authors "A. Writer" --arrangement-name "Default Arrangement" --ccli 123456 --key D --bpm 72 --meter 4/4
  pco songs create --title "Original Song" --authors "A. Writer" --arrangement-name "Default Arrangement" --no-ccli`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		authors, _ := cmd.Flags().GetString("authors")
		arrangementName, _ := cmd.Flags().GetString("arrangement-name")
		noCCLI, _ := cmd.Flags().GetBool("no-ccli")
		key, _ := cmd.Flags().GetString("key")
		meter, _ := cmd.Flags().GetString("meter")
		allowDuplicate, _ := cmd.Flags().GetBool("allow-duplicate")
		resumeSongID, _ := cmd.Flags().GetString("resume-song-id")
		var ccli *int
		if cmd.Flags().Changed("ccli") {
			value, _ := cmd.Flags().GetInt("ccli")
			ccli = &value
		}
		var bpm *float64
		if cmd.Flags().Changed("bpm") {
			value, _ := cmd.Flags().GetFloat64("bpm")
			bpm = &value
		}
		result, err := svc.CreateSong(cmd.Context(), pco.CreateSongOptions{
			Title: title, Authors: authors, ArrangementName: arrangementName, CCLINumber: ccli, NoCCLI: noCCLI,
			Key: key, BPM: bpm, Meter: meter, AllowDuplicate: allowDuplicate, ResumeSongID: resumeSongID,
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			return printer.JSON(result)
		}
		if result.Status == "noop" {
			fmt.Fprintf(printer.Writer(), "No change: song %s (%s) already has arrangement %s (%s); verified\n", result.Song.Title, result.Song.ID, result.Arrangement.Name, result.Arrangement.ID)
		} else {
			fmt.Fprintf(printer.Writer(), "Created song %s (%s) and arrangement %s (%s); verified\n", result.Song.Title, result.Song.ID, result.Arrangement.Name, result.Arrangement.ID)
		}
		for _, warning := range result.Warnings {
			fmt.Fprintf(printer.Writer(), "Warning: %s\n", warning)
		}
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
	songsReplaceCmd.Flags().String("date", "", "service date in YYYY-MM-DD (required unless --plan-id is used)")
	songsReplaceCmd.Flags().String("plan-id", "", "explicit plan ID in the configured service type")
	songsReplaceCmd.Flags().String("current", "", "exact title currently assigned to the plan item")
	songsReplaceCmd.Flags().String("with", "", "exact replacement library-song title")
	songsReplaceCmd.Flags().String("arrangement-id", "", "explicit replacement arrangement ID")
	songsReplaceCmd.Flags().Bool("song-only", false, "leave the replacement arrangement blank")
	songsReplaceCmd.Flags().Bool("dry-run", false, "resolve and validate everything without mutation")
	songsCreateCmd.Flags().String("title", "", "new song title (required)")
	songsCreateCmd.Flags().String("authors", "", "song author or authors (required)")
	songsCreateCmd.Flags().String("arrangement-name", "", "initial arrangement name (required)")
	songsCreateCmd.Flags().Int("ccli", 0, "CCLI song number (required unless --no-ccli)")
	songsCreateCmd.Flags().Bool("no-ccli", false, "confirm that this song has no CCLI number")
	songsCreateCmd.Flags().String("key", "", "optional arrangement chord-chart key")
	songsCreateCmd.Flags().Float64("bpm", 0, "optional arrangement tempo")
	songsCreateCmd.Flags().String("meter", "", "optional arrangement meter, such as 4/4")
	songsCreateCmd.Flags().Bool("allow-duplicate", false, "create despite plausible title or CCLI matches")
	songsCreateCmd.Flags().String("resume-song-id", "", "complete the arrangement for a song left by a partial prior create")

	songsCmd.AddCommand(songsSearchCmd)
	songsCmd.AddCommand(songsHistoryCmd)
	songsCmd.AddCommand(songsArrangementsCmd)
	songsCmd.AddCommand(songsSetCmd)
	songsCmd.AddCommand(songsAddCmd)
	songsCmd.AddCommand(songsReplaceCmd)
	songsCmd.AddCommand(songsCreateCmd)
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
