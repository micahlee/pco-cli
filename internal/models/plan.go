package models

import "encoding/json"

// PlanAttrs holds the attributes of a PCO service plan.
type PlanAttrs struct {
	Title             string `json:"title"`
	Dates             string `json:"dates"`
	SortDate          string `json:"sort_date"`
	PlanNotes         string `json:"plan_notes"`
	PlanningCenterURL string `json:"planning_center_url"`
}

// Plan is a fully resolved plan with ID.
type Plan struct {
	ID    string    `json:"id"`
	Attrs PlanAttrs `json:"attrs"`
}

// PlanItemAttrs holds the attributes of a plan item.
type PlanItemAttrs struct {
	Title                          string          `json:"title"`
	Description                    string          `json:"description"`
	HTMLDetails                    string          `json:"html_details"`
	ItemType                       string          `json:"item_type"`
	Sequence                       int             `json:"sequence"`
	Length                         int             `json:"length"`
	KeyName                        string          `json:"key_name"`
	ServicePosition                string          `json:"service_position"`
	CustomArrangementSequence      json.RawMessage `json:"custom_arrangement_sequence"`
	CustomArrangementSequenceFull  json.RawMessage `json:"custom_arrangement_sequence_full"`
	CustomArrangementSequenceShort json.RawMessage `json:"custom_arrangement_sequence_short"`
	CreatedAt                      string          `json:"created_at"`
	UpdatedAt                      string          `json:"updated_at"`
}

// PlanItem is a resolved plan item with ID and optional relationships.
type PlanItem struct {
	ID            string        `json:"id"`
	SongID        string        `json:"song_id,omitempty"`
	ArrangementID string        `json:"arrangement_id,omitempty"`
	KeyID         string        `json:"key_id,omitempty"`
	MediaIDs      []string      `json:"media_ids,omitempty"`
	Attrs         PlanItemAttrs `json:"attrs"`
}

// PlanExport is a normalized plan document intended for agent workflows.
type PlanExport struct {
	Plan  PlanExportPlan   `json:"plan"`
	Items []PlanExportItem `json:"items"`
	Raw   *PlanExportRaw   `json:"raw,omitempty"`
}

// PlanExportPlan is the top-level plan metadata in an export.
type PlanExportPlan struct {
	ID                string               `json:"id"`
	Title             string               `json:"title"`
	Dates             string               `json:"dates"`
	SortDate          string               `json:"sort_date,omitempty"`
	PlanNotes         string               `json:"plan_notes"`
	PlanNoteDetails   []PlanExportPlanNote `json:"plan_note_details"`
	PlanningCenterURL string               `json:"planning_center_url"`
}

// PlanExportPlanNote is a note attached to the whole service plan.
type PlanExportPlanNote struct {
	ID           string `json:"id"`
	CategoryName string `json:"category_name,omitempty"`
	Content      string `json:"content"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// PlanExportItem is a normalized item in service order.
type PlanExportItem struct {
	ID                             string                 `json:"id"`
	Title                          string                 `json:"title"`
	Type                           string                 `json:"type"`
	Sequence                       int                    `json:"sequence"`
	Length                         int                    `json:"length,omitempty"`
	Description                    string                 `json:"description,omitempty"`
	HTMLDetails                    string                 `json:"html_details,omitempty"`
	ServicePosition                string                 `json:"service_position,omitempty"`
	HeaderContext                  string                 `json:"header_context,omitempty"`
	KeyName                        string                 `json:"key_name,omitempty"`
	CustomArrangementSequence      json.RawMessage        `json:"custom_arrangement_sequence,omitempty"`
	CustomArrangementSequenceFull  json.RawMessage        `json:"custom_arrangement_sequence_full,omitempty"`
	CustomArrangementSequenceShort json.RawMessage        `json:"custom_arrangement_sequence_short,omitempty"`
	Notes                          []PlanExportItemNote   `json:"notes,omitempty"`
	Song                           *PlanExportSong        `json:"song,omitempty"`
	Arrangement                    *PlanExportArrangement `json:"arrangement,omitempty"`
	MediaIDs                       []string               `json:"media_ids,omitempty"`
}

// PlanExportItemNote is a note attached to a plan item.
type PlanExportItemNote struct {
	ID           string `json:"id"`
	CategoryName string `json:"category_name,omitempty"`
	Content      string `json:"content"`
}

// PlanExportSong is normalized song metadata for a song item.
type PlanExportSong struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Author string `json:"author,omitempty"`
}

// PlanExportArrangement is normalized arrangement metadata for a song item.
type PlanExportArrangement struct {
	ID            string          `json:"id"`
	Name          string          `json:"name,omitempty"`
	BPM           *float64        `json:"bpm,omitempty"`
	Meter         string          `json:"meter,omitempty"`
	Length        *int            `json:"length,omitempty"`
	ChordChartKey string          `json:"chord_chart_key,omitempty"`
	Lyrics        string          `json:"lyrics,omitempty"`
	LyricsEnabled *bool           `json:"lyrics_enabled,omitempty"`
	Sequence      []string        `json:"sequence,omitempty"`
	SequenceFull  json.RawMessage `json:"sequence_full,omitempty"`
	SequenceShort []string        `json:"sequence_short,omitempty"`
	Notes         string          `json:"notes,omitempty"`
}

// PlanExportRaw contains raw JSON:API resources for troubleshooting.
type PlanExportRaw struct {
	Plan      *Resource  `json:"plan,omitempty"`
	PlanNotes []Resource `json:"plan_notes,omitempty"`
	Items     []Resource `json:"items,omitempty"`
	Included  []Resource `json:"included,omitempty"`
}

// PlanTemplateAttrs holds the attributes of a plan template.
type PlanTemplateAttrs struct {
	Name string `json:"name"`
}

// PlanTemplate is a resolved plan template.
type PlanTemplate struct {
	ID    string            `json:"id"`
	Attrs PlanTemplateAttrs `json:"attrs"`
}
