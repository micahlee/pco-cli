package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/micahlee/pco-cli/internal/models"
)

// SearchSongs searches for songs by title.
func (s *Service) SearchSongs(ctx context.Context, query string) ([]models.Song, error) {
	params := url.Values{"per_page": {"20"}}
	if query != "" {
		params.Set("where[title]", query)
	}

	data, err := s.Client.Get(ctx, "/services/v2/songs", params)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

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

// SongHistory returns usage data for active songs over the past N weeks.
func (s *Service) SongHistory(ctx context.Context, weeks int) ([]models.SongUsage, int, error) {
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
		dateStr := attrs.SortDate[:10]
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
		json.Unmarshal(p.Attributes, &pattrs)
		dateStr := pattrs.SortDate[:10]

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
		result = append(result, *u)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Dates[0] == result[j].Dates[0] {
			return len(result[i].Dates) > len(result[j].Dates)
		}
		return result[i].Dates[0] > result[j].Dates[0]
	})

	return result, len(plans), nil
}

// SetSong assigns a song to an existing plan item.
func (s *Service) SetSong(ctx context.Context, planID, itemID, songID string) (string, error) {
	song, err := s.GetSong(ctx, songID)
	if err != nil {
		return "", err
	}

	body := fmt.Sprintf(`{
		"data": {
			"type": "Item",
			"attributes": {"title": %q},
			"relationships": {
				"song": {"data": {"type": "Song", "id": %q}}
			}
		}
	}`, song.Attrs.Title, songID)

	_, err = s.Client.Patch(ctx, s.servicePath()+"/plans/"+planID+"/items/"+itemID, body)
	if err != nil {
		return "", err
	}

	return song.Attrs.Title, nil
}

// AddSongItem inserts a new song item after the specified item.
func (s *Service) AddSongItem(ctx context.Context, planID, afterItemID, songID, label string) (string, string, error) {
	// Get anchor item's sequence
	anchorData, err := s.Client.Get(ctx, s.servicePath()+"/plans/"+planID+"/items/"+afterItemID, nil)
	if err != nil {
		return "", "", err
	}
	anchorResource, err := models.ParseOne(anchorData)
	if err != nil {
		return "", "", err
	}
	var anchorAttrs models.PlanItemAttrs
	if err := json.Unmarshal(anchorResource.Attributes, &anchorAttrs); err != nil {
		return "", "", err
	}

	// Get song title
	song, err := s.GetSong(ctx, songID)
	if err != nil {
		return "", "", err
	}

	displayTitle := song.Attrs.Title
	if label != "" {
		displayTitle = fmt.Sprintf("%s (%s)", song.Attrs.Title, label)
	}

	body := fmt.Sprintf(`{
		"data": {
			"type": "Item",
			"attributes": {
				"item_type": "song",
				"title": %q,
				"sequence": %s
			},
			"relationships": {
				"song": {"data": {"type": "Song", "id": %q}}
			}
		}
	}`, displayTitle, strconv.Itoa(anchorAttrs.Sequence+1), songID)

	respData, err := s.Client.Post(ctx, s.servicePath()+"/plans/"+planID+"/items", body)
	if err != nil {
		return "", "", err
	}

	resource, err := models.ParseOne(respData)
	if err != nil {
		return "", "", err
	}

	return displayTitle, resource.ID, nil
}
