package models

// PlanAttrs holds the attributes of a PCO service plan.
type PlanAttrs struct {
	Title           string `json:"title"`
	Dates           string `json:"dates"`
	SortDate        string `json:"sort_date"`
	PlanNotes       string `json:"plan_notes"`
	PlanningCenterURL string `json:"planning_center_url"`
}

// Plan is a fully resolved plan with ID.
type Plan struct {
	ID    string
	Attrs PlanAttrs
}

// PlanItemAttrs holds the attributes of a plan item.
type PlanItemAttrs struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ItemType    string `json:"item_type"`
	Sequence    int    `json:"sequence"`
	Length      int    `json:"length"`
	KeyName     string `json:"key_name"`
}

// PlanItem is a resolved plan item with ID and optional song relationship.
type PlanItem struct {
	ID     string
	SongID string
	Attrs  PlanItemAttrs
}

// PlanTemplateAttrs holds the attributes of a plan template.
type PlanTemplateAttrs struct {
	Name string `json:"name"`
}

// PlanTemplate is a resolved plan template.
type PlanTemplate struct {
	ID    string
	Attrs PlanTemplateAttrs
}
