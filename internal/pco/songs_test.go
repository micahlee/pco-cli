package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/micahlee/pco-cli/internal/api"
	"github.com/micahlee/pco-cli/internal/config"
)

func TestListSongArrangementsIncludesTempoAndMeter(t *testing.T) {
	client := api.New("client", "secret")
	client.HTTPClient = &songFakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/songs/14126642/arrangements?per_page=100": `{
				"data": [
					{
						"type": "Arrangement",
						"id": "20319171",
						"attributes": {
							"name": "Default Arrangement",
							"bpm": 136,
							"meter": "4/4",
							"length": 278,
							"chord_chart_key": "Bb",
							"lyrics": "Verse lyrics\nChorus lyrics",
							"lyrics_enabled": true,
							"sequence": ["Verse 1", "Chorus 1"],
							"sequence_full": [
								{"label": "Verse", "number": "1"},
								{"label": "Chorus", "number": "1"}
							],
							"sequence_short": ["V1", "C1"],
							"notes": "Arrangement note",
							"archived_at": null,
							"updated_at": "2026-06-01T12:00:00Z"
						}
					}
				]
			}`,
		},
	}

	service := &Service{Client: client}
	arrangements, err := service.ListSongArrangements(context.Background(), "14126642")
	if err != nil {
		t.Fatalf("ListSongArrangements returned error: %v", err)
	}

	if len(arrangements) != 1 {
		t.Fatalf("expected 1 arrangement, got %d", len(arrangements))
	}
	arrangement := arrangements[0]
	if arrangement.ID != "20319171" || arrangement.Attrs.Name != "Default Arrangement" {
		t.Fatalf("expected arrangement identity, got %#v", arrangement)
	}
	if arrangement.Attrs.BPM == nil || *arrangement.Attrs.BPM != 136 {
		t.Fatalf("expected BPM 136, got %#v", arrangement.Attrs.BPM)
	}
	if arrangement.Attrs.Meter != "4/4" {
		t.Fatalf("expected meter 4/4, got %q", arrangement.Attrs.Meter)
	}
	if arrangement.Attrs.Length == nil || *arrangement.Attrs.Length != 278 {
		t.Fatalf("expected length 278, got %#v", arrangement.Attrs.Length)
	}
	if arrangement.Attrs.ChordChartKey != "Bb" {
		t.Fatalf("expected chord chart key Bb, got %q", arrangement.Attrs.ChordChartKey)
	}
	if arrangement.Attrs.Lyrics != "Verse lyrics\nChorus lyrics" {
		t.Fatalf("expected lyrics, got %q", arrangement.Attrs.Lyrics)
	}
	if !arrangement.Attrs.LyricsEnabled {
		t.Fatal("expected lyrics enabled")
	}
	if strings.Join(arrangement.Attrs.Sequence, ",") != "Verse 1,Chorus 1" {
		t.Fatalf("expected sequence, got %#v", arrangement.Attrs.Sequence)
	}
	expectedSequenceFull := `[{"label":"Verse","number":"1"},{"label":"Chorus","number":"1"}]`
	if compactJSON(arrangement.Attrs.SequenceFull) != expectedSequenceFull {
		t.Fatalf("expected full sequence, got %s", arrangement.Attrs.SequenceFull)
	}
	if strings.Join(arrangement.Attrs.SequenceShort, ",") != "V1,C1" {
		t.Fatalf("expected short sequence, got %#v", arrangement.Attrs.SequenceShort)
	}
	if arrangement.Attrs.Notes != "Arrangement note" {
		t.Fatalf("expected notes, got %q", arrangement.Attrs.Notes)
	}
	if arrangement.Archived {
		t.Fatal("expected active arrangement")
	}
}

func TestSearchSongsWithArrangementsIncludesArrangementSummaries(t *testing.T) {
	client := api.New("client", "secret")
	client.HTTPClient = &songFakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/songs?per_page=20&where%5Btitle%5D=King+Of+Kings": `{
				"data": [
					{
						"type": "Song",
						"id": "17791889",
						"attributes": {
							"title": "King Of Kings",
							"author": "Brooke Ligertwood",
							"hidden": false
						}
					}
				]
			}`,
			"/services/v2/songs/17791889/arrangements?per_page=100": `{
				"data": [
					{
						"type": "Arrangement",
						"id": "20319171",
						"attributes": {
							"name": "Hillsong Worship",
							"bpm": 136,
							"meter": "4/4"
						}
					}
				]
			}`,
		},
	}

	service := &Service{Client: client}
	results, err := service.SearchSongsWithArrangements(context.Background(), "King Of Kings")
	if err != nil {
		t.Fatalf("SearchSongsWithArrangements returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}
	result := results[0]
	if result.ID != "17791889" || result.Title != "King Of Kings" {
		t.Fatalf("expected song identity, got %#v", result)
	}
	if len(result.Arrangements) != 1 {
		t.Fatalf("expected arrangement summary, got %#v", result.Arrangements)
	}
	arrangement := result.Arrangements[0]
	if arrangement.ID != "20319171" || arrangement.BPM == nil || *arrangement.BPM != 136 || arrangement.Meter != "4/4" {
		t.Fatalf("expected arrangement tempo summary, got %#v", arrangement)
	}
}

func TestSongHistoryIncludesNormalizedJSONFields(t *testing.T) {
	recentDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	olderDate := time.Now().AddDate(0, 0, -14).Format("2006-01-02")

	client := api.New("client", "secret")
	client.HTTPClient = &songFakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/songs?per_page=100": `{
				"data": [
					{
						"type": "Song",
						"id": "song-1",
						"attributes": {"title": "Behold Our God", "hidden": false}
					}
				]
			}`,
			"/services/v2/service_types/643436/plans?filter=past&order=-sort_date&per_page=50": fmt.Sprintf(`{
				"data": [
					{
						"type": "Plan",
						"id": "plan-1",
						"attributes": {"sort_date": %q}
					},
					{
						"type": "Plan",
						"id": "plan-2",
						"attributes": {"sort_date": %q}
					}
				]
			}`, recentDate+"T09:00:00Z", olderDate+"T09:00:00Z"),
			"/services/v2/service_types/643436/plans/plan-1/items?filter=songs&per_page=50": `{
				"data": [
					{
						"type": "Item",
						"id": "item-1",
						"attributes": {"title": "Behold Our God", "item_type": "song", "sequence": 1},
						"relationships": {"song": {"data": {"type": "Song", "id": "song-1"}}}
					}
				]
			}`,
			"/services/v2/service_types/643436/plans/plan-2/items?filter=songs&per_page=50": `{
				"data": [
					{
						"type": "Item",
						"id": "item-2",
						"attributes": {"title": "Behold Our God", "item_type": "song", "sequence": 1},
						"relationships": {"song": {"data": {"type": "Song", "id": "song-1"}}}
					}
				]
			}`,
		},
	}

	service := &Service{
		Client: client,
		Config: &config.Config{ServiceTypeID: "643436"},
	}

	usage, planCount, err := service.SongHistory(context.Background(), 20)
	if err != nil {
		t.Fatalf("SongHistory returned error: %v", err)
	}

	if planCount != 2 {
		t.Fatalf("expected 2 plans, got %d", planCount)
	}
	if len(usage) != 1 {
		t.Fatalf("expected 1 song usage, got %#v", usage)
	}
	if usage[0].SongID != "song-1" || usage[0].Title != "Behold Our God" {
		t.Fatalf("expected song identity, got %#v", usage[0])
	}
	if usage[0].Uses != 2 {
		t.Fatalf("expected uses 2, got %d", usage[0].Uses)
	}
	if usage[0].LastUsed != recentDate {
		t.Fatalf("expected last used %s, got %s", recentDate, usage[0].LastUsed)
	}
	if strings.Join(usage[0].Dates, ",") != recentDate+","+olderDate {
		t.Fatalf("expected sorted dates, got %#v", usage[0].Dates)
	}

	data, err := json.Marshal(usage[0])
	if err != nil {
		t.Fatalf("marshaling usage: %v", err)
	}
	var jsonShape map[string]any
	if err := json.Unmarshal(data, &jsonShape); err != nil {
		t.Fatalf("unmarshaling usage JSON: %v", err)
	}
	for _, key := range []string{"song_id", "title", "uses", "last_used", "dates"} {
		if _, ok := jsonShape[key]; !ok {
			t.Fatalf("expected JSON key %q in %s", key, data)
		}
	}
	for _, key := range []string{"SongID", "Title", "Uses", "LastUsed", "Dates"} {
		if _, ok := jsonShape[key]; ok {
			t.Fatalf("did not expect Go-style JSON key %q in %s", key, data)
		}
	}
}

func TestSetSongAttachesDefaultArrangement(t *testing.T) {
	client := api.New("client", "secret")
	fake := &songFakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/songs/song-1": `{
				"data": {
					"type": "Song",
					"id": "song-1",
					"attributes": {"title": "Rejoice"}
				}
			}`,
			"/services/v2/songs/song-1/arrangements?per_page=100": `{
				"data": [
					{
						"type": "Arrangement",
						"id": "arr-1",
						"attributes": {"name": "Default Arrangement", "archived_at": null}
					}
				]
			}`,
			"/services/v2/service_types/643436/plans/plan-1/items/item-1": `{
				"data": {"type": "Item", "id": "item-1"}
			}`,
		},
	}
	client.HTTPClient = fake

	service := &Service{
		Client: client,
		Config: &config.Config{ServiceTypeID: "643436"},
	}

	result, err := service.SetSong(context.Background(), "plan-1", "item-1", "song-1", SongAssignmentOptions{})
	if err != nil {
		t.Fatalf("SetSong returned error: %v", err)
	}

	if result.Title != "Rejoice" || result.ArrangementID != "arr-1" || result.ArrangementName != "Default Arrangement" {
		t.Fatalf("expected song and arrangement result, got %#v", result)
	}

	body := fake.requestJSON(t, http.MethodPatch, "/services/v2/service_types/643436/plans/plan-1/items/item-1")
	data := body["data"].(map[string]any)
	relationships := data["relationships"].(map[string]any)
	assertRelationship(t, relationships, "song", "Song", "song-1")
	assertRelationship(t, relationships, "arrangement", "Arrangement", "arr-1")
}

func TestSetSongOnlyLeavesArrangementBlank(t *testing.T) {
	client := api.New("client", "secret")
	fake := &songFakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/songs/song-1": `{
				"data": {
					"type": "Song",
					"id": "song-1",
					"attributes": {"title": "Rejoice"}
				}
			}`,
			"/services/v2/service_types/643436/plans/plan-1/items/item-1": `{
				"data": {"type": "Item", "id": "item-1"}
			}`,
		},
	}
	client.HTTPClient = fake

	service := &Service{
		Client: client,
		Config: &config.Config{ServiceTypeID: "643436"},
	}

	result, err := service.SetSong(context.Background(), "plan-1", "item-1", "song-1", SongAssignmentOptions{SongOnly: true})
	if err != nil {
		t.Fatalf("SetSong returned error: %v", err)
	}

	if result.ArrangementID != "" || !strings.Contains(result.Warning, "no arrangement attached") {
		t.Fatalf("expected song-only warning without arrangement, got %#v", result)
	}

	body := fake.requestJSON(t, http.MethodPatch, "/services/v2/service_types/643436/plans/plan-1/items/item-1")
	data := body["data"].(map[string]any)
	relationships := data["relationships"].(map[string]any)
	assertRelationship(t, relationships, "song", "Song", "song-1")
	arrangement := relationships["arrangement"].(map[string]any)
	if arrangement["data"] != nil {
		t.Fatalf("expected null arrangement relationship, got %#v", arrangement)
	}
}

func TestAddSongItemAttachesExplicitArrangement(t *testing.T) {
	client := api.New("client", "secret")
	fake := &songFakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/service_types/643436/plans/plan-1/items/anchor-1": `{
				"data": {
					"type": "Item",
					"id": "anchor-1",
					"attributes": {"title": "Anchor", "sequence": 4}
				}
			}`,
			"/services/v2/songs/song-1": `{
				"data": {
					"type": "Song",
					"id": "song-1",
					"attributes": {"title": "Come Thou Fount"}
				}
			}`,
			"/services/v2/songs/song-1/arrangements?per_page=100": `{
				"data": [
					{
						"type": "Arrangement",
						"id": "arr-1",
						"attributes": {"name": "Default Arrangement", "archived_at": null}
					},
					{
						"type": "Arrangement",
						"id": "arr-2",
						"attributes": {"name": "Acoustic", "archived_at": null}
					}
				]
			}`,
			"/services/v2/service_types/643436/plans/plan-1/items": `{
				"data": {"type": "Item", "id": "new-1"}
			}`,
		},
	}
	client.HTTPClient = fake

	service := &Service{
		Client: client,
		Config: &config.Config{ServiceTypeID: "643436"},
	}

	result, err := service.AddSongItem(context.Background(), "plan-1", "anchor-1", "song-1", "Lord's Supper", SongAssignmentOptions{ArrangementID: "arr-2"})
	if err != nil {
		t.Fatalf("AddSongItem returned error: %v", err)
	}

	if result.ItemID != "new-1" || result.Title != "Come Thou Fount (Lord's Supper)" || result.ArrangementID != "arr-2" {
		t.Fatalf("expected new song item result, got %#v", result)
	}

	body := fake.requestJSON(t, http.MethodPost, "/services/v2/service_types/643436/plans/plan-1/items")
	data := body["data"].(map[string]any)
	attrs := data["attributes"].(map[string]any)
	if attrs["sequence"].(float64) != 5 {
		t.Fatalf("expected sequence 5, got %#v", attrs["sequence"])
	}
	if attrs["title"] != "Come Thou Fount (Lord's Supper)" {
		t.Fatalf("expected labeled title, got %#v", attrs["title"])
	}
	relationships := data["relationships"].(map[string]any)
	assertRelationship(t, relationships, "song", "Song", "song-1")
	assertRelationship(t, relationships, "arrangement", "Arrangement", "arr-2")
}

func TestSetSongRequiresArrangementIDWhenMultipleActiveArrangements(t *testing.T) {
	client := api.New("client", "secret")
	client.HTTPClient = &songFakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/songs/song-1": `{
				"data": {
					"type": "Song",
					"id": "song-1",
					"attributes": {"title": "Ambiguous Song"}
				}
			}`,
			"/services/v2/songs/song-1/arrangements?per_page=100": `{
				"data": [
					{
						"type": "Arrangement",
						"id": "arr-1",
						"attributes": {"name": "Acoustic", "archived_at": null}
					},
					{
						"type": "Arrangement",
						"id": "arr-2",
						"attributes": {"name": "Full Band", "archived_at": null}
					}
				]
			}`,
		},
	}

	service := &Service{
		Client: client,
		Config: &config.Config{ServiceTypeID: "643436"},
	}

	_, err := service.SetSong(context.Background(), "plan-1", "item-1", "song-1", SongAssignmentOptions{})
	if err == nil {
		t.Fatal("expected ambiguous arrangement error")
	}
	if !strings.Contains(err.Error(), "multiple active arrangements") || !strings.Contains(err.Error(), "--arrangement-id") {
		t.Fatalf("expected actionable arrangement error, got %v", err)
	}
}

type songFakeHTTPClient struct {
	t         *testing.T
	responses map[string]string
	requests  []songFakeRequest
}

type songFakeRequest struct {
	method string
	path   string
	body   string
}

func (c *songFakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	key := req.URL.Path
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.RawQuery
	}
	var requestBody string
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			c.t.Fatalf("reading request body: %v", err)
		}
		requestBody = string(data)
	}
	c.requests = append(c.requests, songFakeRequest{
		method: req.Method,
		path:   req.URL.Path,
		body:   requestBody,
	})
	body, ok := c.responses[key]
	if !ok {
		c.t.Fatalf("unexpected request path: %s", req.URL.String())
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func (c *songFakeHTTPClient) requestJSON(t *testing.T, method, path string) map[string]any {
	t.Helper()
	idx := slices.IndexFunc(c.requests, func(req songFakeRequest) bool {
		return req.method == method && req.path == path
	})
	if idx == -1 {
		t.Fatalf("expected %s %s request in %#v", method, path, c.requests)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(c.requests[idx].body), &body); err != nil {
		t.Fatalf("unmarshaling request body %q: %v", c.requests[idx].body, err)
	}
	return body
}

func assertRelationship(t *testing.T, relationships map[string]any, name, wantType, wantID string) {
	t.Helper()
	rel := relationships[name].(map[string]any)
	data := rel["data"].(map[string]any)
	if data["type"] != wantType || data["id"] != wantID {
		t.Fatalf("expected %s relationship %s/%s, got %#v", name, wantType, wantID, data)
	}
}
