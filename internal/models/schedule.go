package models

// ScheduleAttrs holds the attributes of a serve schedule entry.
type ScheduleAttrs struct {
	SortDate         string `json:"sort_date"`
	Status           string `json:"status"`
	TeamPositionName string `json:"team_position_name"`
}

// Schedule is a fully resolved schedule entry (serve request).
type Schedule struct {
	ID    string
	Attrs ScheduleAttrs
}
