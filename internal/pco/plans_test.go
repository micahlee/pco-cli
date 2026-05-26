package pco

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/micahlee/pco-cli/internal/api"
	"github.com/micahlee/pco-cli/internal/config"
)

func TestExportPlanIncludesDetailsAndIncludedResources(t *testing.T) {
	client := api.New("client", "secret")
	client.HTTPClient = &fakeHTTPClient{
		t: t,
		responses: map[string]string{
			"/services/v2/service_types/643436/plans/87689985": `{
				"data": {
					"type": "Plan",
					"id": "87689985",
					"attributes": {
						"title": "May 31 Worship",
						"dates": "May 31, 2026",
						"sort_date": "2026-05-31T09:00:00Z",
						"plan_notes": "Plan-level note",
						"planning_center_url": "https://services.planningcenteronline.com/plans/87689985"
					}
				}
			}`,
			"/services/v2/service_types/643436/plans/87689985/items": `{
				"data": [
					{
						"type": "Item",
						"id": "song-1",
						"attributes": {
							"title": "Behold Our God",
							"description": "",
							"html_details": "",
							"item_type": "song",
							"sequence": 3,
							"length": 240,
							"key_name": "A",
							"service_position": "during"
						},
						"relationships": {
							"song": {"data": {"type": "Song", "id": "14126642"}},
							"arrangement": {"data": {"type": "Arrangement", "id": "arr-1"}},
							"media": {"data": [{"type": "Media", "id": "media-1"}]}
						}
					},
					{
						"type": "Item",
						"id": "header-1",
						"attributes": {
							"title": "Word",
							"item_type": "header",
							"sequence": 1,
							"service_position": "during"
						}
					},
					{
						"type": "Item",
						"id": "reading-1",
						"attributes": {
							"title": "Reading of Sermon Passage",
							"description": "Galatians 1:1-5",
							"html_details": "<p>Galatians 1:1-5</p>",
							"item_type": "item",
							"sequence": 2,
							"service_position": "pre"
						}
					}
				],
				"included": [
					{
						"type": "Song",
						"id": "14126642",
						"attributes": {"title": "Behold Our God", "author": "Sovereign Grace"}
					},
					{
						"type": "Arrangement",
						"id": "arr-1",
						"attributes": {"name": "Congregational"}
					},
					{
						"type": "ItemNote",
						"id": "note-1",
						"attributes": {"category_name": "Scripture", "content": "Read slowly."},
						"relationships": {"item": {"data": {"type": "Item", "id": "reading-1"}}}
					}
				]
			}`,
		},
	}

	service := &Service{
		Client: client,
		Config: &config.Config{ServiceTypeID: "643436"},
	}

	export, err := service.ExportPlan(context.Background(), "87689985", true)
	if err != nil {
		t.Fatalf("ExportPlan returned error: %v", err)
	}

	if export.Plan.Title != "May 31 Worship" {
		t.Fatalf("expected plan title, got %q", export.Plan.Title)
	}
	if len(export.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(export.Items))
	}

	if export.Items[0].ID != "header-1" || export.Items[1].ID != "reading-1" || export.Items[2].ID != "song-1" {
		t.Fatalf("items were not sorted by sequence: %#v", export.Items)
	}

	reading := export.Items[1]
	if reading.Description != "Galatians 1:1-5" {
		t.Fatalf("expected reading description, got %q", reading.Description)
	}
	if reading.HTMLDetails != "<p>Galatians 1:1-5</p>" {
		t.Fatalf("expected reading html details, got %q", reading.HTMLDetails)
	}
	if reading.HeaderContext != "Word" {
		t.Fatalf("expected header context Word, got %q", reading.HeaderContext)
	}
	if len(reading.Notes) != 1 || reading.Notes[0].Content != "Read slowly." {
		t.Fatalf("expected item note, got %#v", reading.Notes)
	}

	song := export.Items[2]
	if song.Song == nil || song.Song.ID != "14126642" || song.Song.Title != "Behold Our God" {
		t.Fatalf("expected song metadata, got %#v", song.Song)
	}
	if song.Arrangement == nil || song.Arrangement.ID != "arr-1" || song.Arrangement.Name != "Congregational" {
		t.Fatalf("expected arrangement metadata, got %#v", song.Arrangement)
	}
	if len(song.MediaIDs) != 1 || song.MediaIDs[0] != "media-1" {
		t.Fatalf("expected media IDs, got %#v", song.MediaIDs)
	}

	if export.Raw == nil || export.Raw.Plan == nil || len(export.Raw.Items) != 3 || len(export.Raw.Included) != 3 {
		t.Fatalf("expected raw JSON:API resources, got %#v", export.Raw)
	}
}

type fakeHTTPClient struct {
	t         *testing.T
	responses map[string]string
}

func (c *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/services/v2/service_types/643436/plans/87689985/items" {
		if got := req.URL.Query().Get("include"); got != "song,arrangement,item_notes,media" {
			c.t.Fatalf("unexpected include query: %q", got)
		}
		if got := req.URL.Query().Get("per_page"); got != "100" {
			c.t.Fatalf("unexpected per_page query: %q", got)
		}
	}

	body, ok := c.responses[req.URL.Path]
	if !ok {
		c.t.Fatalf("unexpected request path: %s", req.URL.String())
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}
