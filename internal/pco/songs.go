package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/micahlee/pco-cli/internal/models"
)

// SongAssignmentOptions controls how a song item relationship is updated.
type SongAssignmentOptions struct {
	ArrangementID string
	SongOnly      bool
}

// SongAssignmentResult describes a song item mutation.
type SongAssignmentResult struct {
	ItemID          string `json:"item_id"`
	SongID          string `json:"song_id"`
	Title           string `json:"title"`
	ArrangementID   string `json:"arrangement_id,omitempty"`
	ArrangementName string `json:"arrangement_name,omitempty"`
	Warning         string `json:"warning,omitempty"`
}

// SearchSongs searches for songs by title.
func (s *Service) SearchSongs(ctx context.Context, query string) ([]models.Song, error) {
	params := url.Values{"per_page": {"100"}}
	if query != "" {
		params.Set("where[title]", query)
	}

	resources, err := s.Client.GetAll(ctx, "/services/v2/songs", params)
	if err != nil {
		return nil, err
	}
	return songsFromResources(resources)
}

func songsFromResources(resources []models.Resource) ([]models.Song, error) {
	songs := make([]models.Song, len(resources))
	for i, r := range resources {
		var attrs models.SongAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		songs[i] = models.Song{ID: r.ID, Attrs: attrs}
	}
	return songs, nil
}

// SearchSongsWithArrangements searches for songs and includes arrangement summaries.
func (s *Service) SearchSongsWithArrangements(ctx context.Context, query string) ([]models.SongSearchResult, error) {
	songs, err := s.SearchSongs(ctx, query)
	if err != nil {
		return nil, err
	}

	results := make([]models.SongSearchResult, len(songs))
	for i, song := range songs {
		arrangements, err := s.ListSongArrangements(ctx, song.ID)
		if err != nil {
			return nil, err
		}
		results[i] = models.SongSearchResult{
			ID:           song.ID,
			Title:        song.Attrs.Title,
			Author:       song.Attrs.Author,
			Arrangements: arrangementSummaries(arrangements),
		}
	}
	return results, nil
}

func arrangementSummaries(arrangements []models.Arrangement) []models.ArrangementSummary {
	summaries := make([]models.ArrangementSummary, len(arrangements))
	for i, arrangement := range arrangements {
		summaries[i] = models.ArrangementSummary{
			ID:            arrangement.ID,
			Name:          arrangement.Attrs.Name,
			BPM:           arrangement.Attrs.BPM,
			Meter:         arrangement.Attrs.Meter,
			Length:        arrangement.Attrs.Length,
			ChordChartKey: arrangement.Attrs.ChordChartKey,
			Archived:      arrangement.Archived,
			UpdatedAt:     arrangement.Attrs.UpdatedAt,
		}
	}
	return summaries
}

// GetSong returns a single song by ID.
func (s *Service) GetSong(ctx context.Context, songID string) (*models.Song, error) {
	data, err := s.Client.Get(ctx, "/services/v2/songs/"+songID, nil)
	if err != nil {
		return nil, err
	}

	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}

	var attrs models.SongAttrs
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return nil, err
	}

	return &models.Song{ID: resource.ID, Attrs: attrs}, nil
}

// ListSongArrangements returns all arrangements for a song.
func (s *Service) ListSongArrangements(ctx context.Context, songID string) ([]models.Arrangement, error) {
	resources, err := s.Client.GetAll(ctx, "/services/v2/songs/"+songID+"/arrangements", url.Values{"per_page": {"100"}})
	if err != nil {
		return nil, err
	}

	arrangements := make([]models.Arrangement, len(resources))
	for i, r := range resources {
		var attrs models.ArrangementAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		arrangements[i] = models.Arrangement{
			ID:       r.ID,
			Attrs:    attrs,
			Archived: attrs.ArchivedAt != "",
		}
	}
	return arrangements, nil
}

// SongHistory returns usage data for active songs over the past N weeks.
func (s *Service) SongHistory(ctx context.Context, weeks int) ([]models.SongUsage, int, error) {
	if weeks <= 0 {
		return nil, 0, fmt.Errorf("weeks must be greater than zero")
	}
	cutoff := time.Now().AddDate(0, 0, -weeks*7)

	// Fetch all active songs
	activeSongs := make(map[string]string) // id -> title
	for r, err := range s.Client.PageIterator(ctx, "/services/v2/songs", url.Values{"per_page": {"100"}}) {
		if err != nil {
			return nil, 0, err
		}
		var attrs models.SongAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, 0, err
		}
		if !attrs.Hidden {
			activeSongs[r.ID] = attrs.Title
		}
	}

	// Fetch past plans within the window
	var plans []models.Resource
	for r, err := range s.Client.PageIterator(ctx, s.servicePath()+"/plans", url.Values{
		"filter": {"past"}, "per_page": {"50"}, "order": {"-sort_date"},
	}) {
		if err != nil {
			return nil, 0, err
		}
		var attrs models.PlanAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, 0, err
		}
		dateStr, dateErr := datePart(attrs.SortDate)
		if dateErr != nil {
			return nil, 0, fmt.Errorf("plan %s has malformed or missing sort_date: %w", r.ID, dateErr)
		}
		planDate, _ := time.Parse("2006-01-02", dateStr)
		if planDate.Before(cutoff) {
			break
		}
		plans = append(plans, r)
	}

	// For each plan, fetch songs
	usage := make(map[string]*models.SongUsage)
	for _, p := range plans {
		var pattrs models.PlanAttrs
		if err := json.Unmarshal(p.Attributes, &pattrs); err != nil {
			return nil, 0, fmt.Errorf("decoding plan %s: %w", p.ID, err)
		}
		dateStr, dateErr := datePart(pattrs.SortDate)
		if dateErr != nil {
			return nil, 0, fmt.Errorf("plan %s has malformed or missing sort_date: %w", p.ID, dateErr)
		}

		items, err := s.ListPlanSongs(ctx, p.ID)
		if err != nil {
			return nil, 0, err
		}

		for _, item := range items {
			if item.SongID == "" || activeSongs[item.SongID] == "" {
				continue
			}
			if _, ok := usage[item.SongID]; !ok {
				usage[item.SongID] = &models.SongUsage{
					SongID: item.SongID,
					Title:  activeSongs[item.SongID],
				}
			}
			usage[item.SongID].Dates = append(usage[item.SongID].Dates, dateStr)
		}
	}

	// Sort by most recently used, then by frequency
	result := make([]models.SongUsage, 0, len(usage))
	for _, u := range usage {
		sort.Sort(sort.Reverse(sort.StringSlice(u.Dates)))
		u.Uses = len(u.Dates)
		u.LastUsed = u.Dates[0]
		result = append(result, *u)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LastUsed == result[j].LastUsed {
			return result[i].Uses > result[j].Uses
		}
		return result[i].LastUsed > result[j].LastUsed
	})

	return result, len(plans), nil
}

// SetSong assigns a song to an existing plan item.
func (s *Service) SetSong(ctx context.Context, planID, itemID, songID string, opts SongAssignmentOptions) (*SongAssignmentResult, error) {
	if strings.TrimSpace(planID) == "" || strings.TrimSpace(itemID) == "" || strings.TrimSpace(songID) == "" {
		return nil, fmt.Errorf("plan ID, item ID, and song ID are required")
	}
	if _, err := s.GetPlan(ctx, planID); err != nil {
		return nil, fmt.Errorf("preflight plan %s: %w", planID, err)
	}
	if _, err := s.GetPlanItem(ctx, planID, itemID); err != nil {
		return nil, fmt.Errorf("preflight item %s in plan %s: %w", itemID, planID, err)
	}
	song, err := s.GetSong(ctx, songID)
	if err != nil {
		return nil, err
	}
	arrangement, err := s.resolveSongArrangement(ctx, songID, opts)
	if err != nil {
		return nil, err
	}

	if err := s.patchSongItem(ctx, planID, itemID, *song, arrangement, opts.SongOnly); err != nil {
		return nil, err
	}
	verified, err := s.GetPlanItem(ctx, planID, itemID)
	if err != nil {
		return nil, fmt.Errorf("song assignment request succeeded but verification read failed: %w", err)
	}
	wantArrangement := ""
	if arrangement != nil {
		wantArrangement = arrangement.ID
	}
	if verified.SongID != songID || verified.ArrangementID != wantArrangement {
		return nil, fmt.Errorf("song assignment verification failed: item has song %q and arrangement %q; expected %q and %q", verified.SongID, verified.ArrangementID, songID, wantArrangement)
	}
	return songAssignmentResult(itemID, songID, song.Attrs.Title, arrangement, opts.SongOnly), nil
}

func (s *Service) patchSongItem(ctx context.Context, planID, itemID string, song models.Song, arrangement *models.Arrangement, songOnly bool) error {
	relationships := songItemRelationships(song.ID, arrangement)
	if songOnly {
		relationships["arrangement"] = map[string]any{"data": nil}
	}
	body, err := json.Marshal(map[string]any{"data": map[string]any{
		"type": "Item", "id": itemID, "attributes": map[string]any{"title": song.Attrs.Title}, "relationships": relationships,
	}})
	if err != nil {
		return err
	}
	_, err = s.Client.Patch(ctx, s.servicePath()+"/plans/"+planID+"/items/"+itemID, string(body))
	return err
}

// AddSongItem inserts a new song item after the specified item.
func (s *Service) AddSongItem(ctx context.Context, planID, afterItemID, songID, label string, opts SongAssignmentOptions) (*SongAssignmentResult, error) {
	if strings.TrimSpace(planID) == "" || strings.TrimSpace(afterItemID) == "" || strings.TrimSpace(songID) == "" {
		return nil, fmt.Errorf("plan ID, anchor item ID, and song ID are required")
	}
	if _, err := s.GetPlan(ctx, planID); err != nil {
		return nil, fmt.Errorf("preflight plan %s: %w", planID, err)
	}
	// Get anchor item's sequence
	anchorData, err := s.Client.Get(ctx, s.servicePath()+"/plans/"+planID+"/items/"+afterItemID, nil)
	if err != nil {
		return nil, err
	}
	anchorResource, err := models.ParseOne(anchorData)
	if err != nil {
		return nil, err
	}
	var anchorAttrs models.PlanItemAttrs
	if err := json.Unmarshal(anchorResource.Attributes, &anchorAttrs); err != nil {
		return nil, err
	}

	// Get song title
	song, err := s.GetSong(ctx, songID)
	if err != nil {
		return nil, err
	}
	arrangement, err := s.resolveSongArrangement(ctx, songID, opts)
	if err != nil {
		return nil, err
	}

	displayTitle := song.Attrs.Title
	if label != "" {
		displayTitle = fmt.Sprintf("%s (%s)", song.Attrs.Title, label)
	}

	body, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"type": "Item",
			"attributes": map[string]any{
				"item_type": "song",
				"title":     displayTitle,
				"sequence":  anchorAttrs.Sequence + 1,
			},
			"relationships": songItemRelationships(songID, arrangement),
		},
	})
	if err != nil {
		return nil, err
	}

	respData, err := s.Client.Post(ctx, s.servicePath()+"/plans/"+planID+"/items", string(body))
	if err != nil {
		return nil, err
	}

	resource, err := models.ParseOne(respData)
	if err != nil {
		return nil, err
	}

	verified, err := s.GetPlanItem(ctx, planID, resource.ID)
	if err != nil {
		return nil, fmt.Errorf("song item %s was created but verification read failed: %w", resource.ID, err)
	}
	wantArrangement := ""
	if arrangement != nil {
		wantArrangement = arrangement.ID
	}
	if verified.SongID != songID || verified.ArrangementID != wantArrangement {
		return nil, fmt.Errorf("song item %s was created but verification failed: item has song %q and arrangement %q; expected %q and %q", resource.ID, verified.SongID, verified.ArrangementID, songID, wantArrangement)
	}
	return songAssignmentResult(resource.ID, songID, displayTitle, arrangement, opts.SongOnly), nil
}

func (s *Service) resolveSongArrangement(ctx context.Context, songID string, opts SongAssignmentOptions) (*models.Arrangement, error) {
	if opts.SongOnly {
		return nil, nil
	}

	arrangements, err := s.ListSongArrangements(ctx, songID)
	if err != nil {
		return nil, err
	}

	if opts.ArrangementID != "" {
		for _, arrangement := range arrangements {
			if arrangement.ID == opts.ArrangementID {
				return &arrangement, nil
			}
		}
		return nil, fmt.Errorf("arrangement %s not found for song %s", opts.ArrangementID, songID)
	}

	var active []models.Arrangement
	var defaults []models.Arrangement
	for _, arrangement := range arrangements {
		if arrangement.Archived {
			continue
		}
		active = append(active, arrangement)
		if strings.EqualFold(strings.TrimSpace(arrangement.Attrs.Name), "Default Arrangement") {
			defaults = append(defaults, arrangement)
		}
	}
	if len(defaults) == 1 {
		return &defaults[0], nil
	}
	if len(defaults) > 1 {
		return nil, fmt.Errorf("song %s has multiple active arrangements named Default Arrangement (%s); pass --arrangement-id", songID, arrangementChoices(defaults))
	}

	if len(active) == 1 {
		return &active[0], nil
	}
	if len(active) == 0 {
		return nil, fmt.Errorf("song %s has no active arrangements; pass --song-only to leave arrangement blank", songID)
	}

	return nil, fmt.Errorf("song %s has multiple active arrangements (%s); pass --arrangement-id", songID, arrangementChoices(active))
}

func arrangementChoices(arrangements []models.Arrangement) string {
	choices := make([]string, len(arrangements))
	for i, arrangement := range arrangements {
		name := arrangement.Attrs.Name
		if name == "" {
			name = "unnamed"
		}
		choices[i] = fmt.Sprintf("%s %q", arrangement.ID, name)
	}
	return strings.Join(choices, ", ")
}

func songItemRelationships(songID string, arrangement *models.Arrangement) map[string]any {
	relationships := map[string]any{
		"song": map[string]any{
			"data": map[string]string{"type": "Song", "id": songID},
		},
	}
	if arrangement != nil {
		relationships["arrangement"] = map[string]any{
			"data": map[string]string{"type": "Arrangement", "id": arrangement.ID},
		}
	}
	return relationships
}

func songAssignmentResult(itemID, songID, title string, arrangement *models.Arrangement, songOnly bool) *SongAssignmentResult {
	result := &SongAssignmentResult{
		ItemID: itemID,
		SongID: songID,
		Title:  title,
	}
	if arrangement != nil {
		result.ArrangementID = arrangement.ID
		result.ArrangementName = arrangement.Attrs.Name
		return result
	}
	if songOnly {
		result.Warning = "no arrangement attached (--song-only)"
	}
	return result
}
