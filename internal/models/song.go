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
