package cmd

import (
	"fmt"
	"strconv"
	"strings"

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

		fmt.Fprintf(printer.Writer(), "Song history — past %d weeks (active songs only, %d plans)\n\n", weeks, planCount)

		headers := []string{"Song", "Uses", "Last Used", "All Dates"}
		rows := make([][]string, len(usage))
		for i, u := range usage {
			rows[i] = []string{
				u.Title,
				strconv.Itoa(len(u.Dates)),
				u.Dates[0],
				strings.Join(u.Dates, ", "),
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
		title, err := svc.SetSong(cmd.Context(), args[0], args[1], args[2])
		if err != nil {
			return err
		}
		fmt.Fprintf(printer.Writer(), "Set '%s' on item %s (plan %s)\n", title, args[1], args[0])
		return nil
	},
}

var songsAddCmd = &cobra.Command{
	Use:   "add <plan-id> <after-item-id> <song-id>",
	Short: "Insert a new song item after the specified item",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		title, newID, err := svc.AddSongItem(cmd.Context(), args[0], args[1], args[2], label)
		if err != nil {
			return err
		}
		fmt.Fprintf(printer.Writer(), "Added '%s' after item %s (new item ID: %s)\n", title, args[1], newID)
		return nil
	},
}

func init() {
	songsSearchCmd.Flags().String("query", "", "search by song title")
	songsHistoryCmd.Flags().Int("weeks", 16, "number of weeks to look back")
	songsAddCmd.Flags().String("label", "", "label to append in parentheses (e.g. \"Lord's Supper\")")

	songsCmd.AddCommand(songsSearchCmd)
	songsCmd.AddCommand(songsHistoryCmd)
	songsCmd.AddCommand(songsSetCmd)
	songsCmd.AddCommand(songsAddCmd)
	rootCmd.AddCommand(songsCmd)
}
