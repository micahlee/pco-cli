package models

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

// SongUsage tracks how a song has been used in past plans.
type SongUsage struct {
	SongID string
	Title  string
	Dates  []string
}

// ArrangementAttrs holds the useful planning attributes of a song arrangement.
type ArrangementAttrs struct {
	Name          string   `json:"name"`
	BPM           *float64 `json:"bpm"`
	Meter         string   `json:"meter"`
	Length        *int     `json:"length"`
	ChordChartKey string   `json:"chord_chart_key"`
	ArchivedAt    string   `json:"archived_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// Arrangement is a resolved song arrangement with ID.
type Arrangement struct {
	ID       string           `json:"id"`
	Attrs    ArrangementAttrs `json:"attrs"`
	Archived bool             `json:"archived"`
}
