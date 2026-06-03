package pco

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/micahlee/pco-cli/internal/api"
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

type songFakeHTTPClient struct {
	t         *testing.T
	responses map[string]string
}

func (c *songFakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	key := req.URL.Path
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.RawQuery
	}
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
