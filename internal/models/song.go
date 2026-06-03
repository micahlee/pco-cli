package models

import "encoding/json"

// SongAttrs holds the attributes of a PCO song.
type SongAttrs struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Hidden bool   `json:"hidden"`
}

// Song is a fully resolved song with ID.
type Song struct {
	ID    string
	Attrs SongAttrs
}

// SongSearchResult is a song search hit with optional arrangement summaries.
type SongSearchResult struct {
	ID           string               `json:"id"`
	Title        string               `json:"title"`
	Author       string               `json:"author"`
	Arrangements []ArrangementSummary `json:"arrangements"`
}

// ArrangementSummary is the compact arrangement data included in song search results.
type ArrangementSummary struct {
	ID            string   `json:"id"`
	Name          string   `json:"name,omitempty"`
	BPM           *float64 `json:"bpm,omitempty"`
	Meter         string   `json:"meter,omitempty"`
	Length        *int     `json:"length,omitempty"`
	ChordChartKey string   `json:"chord_chart_key,omitempty"`
	Archived      bool     `json:"archived"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

// SongUsage tracks how a song has been used in past plans.
type SongUsage struct {
	SongID   string   `json:"song_id"`
	Title    string   `json:"title"`
	Uses     int      `json:"uses"`
	LastUsed string   `json:"last_used"`
	Dates    []string `json:"dates"`
}

// ArrangementAttrs holds the useful planning attributes of a song arrangement.
type ArrangementAttrs struct {
	Name          string          `json:"name"`
	BPM           *float64        `json:"bpm"`
	Meter         string          `json:"meter"`
	Length        *int            `json:"length"`
	ChordChartKey string          `json:"chord_chart_key"`
	Lyrics        string          `json:"lyrics"`
	LyricsEnabled bool            `json:"lyrics_enabled"`
	Sequence      []string        `json:"sequence"`
	SequenceFull  json.RawMessage `json:"sequence_full"`
	SequenceShort []string        `json:"sequence_short"`
	Notes         string          `json:"notes"`
	ArchivedAt    string          `json:"archived_at"`
	UpdatedAt     string          `json:"updated_at"`
}

// Arrangement is a resolved song arrangement with ID.
type Arrangement struct {
	ID       string           `json:"id"`
	Attrs    ArrangementAttrs `json:"attrs"`
	Archived bool             `json:"archived"`
}
