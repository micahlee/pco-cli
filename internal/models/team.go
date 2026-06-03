package models

import "encoding/json"

// TeamPersonAttrs holds attributes for a team person (from team people endpoint).
type TeamPersonAttrs struct {
	FullName string `json:"full_name"`
}

// TeamPerson is a person on a team.
type TeamPerson struct {
	ID    string
	Attrs TeamPersonAttrs
}

// BandMember combines a team person with their typical positions.
type BandMember struct {
	PersonID         string
	Name             string
	TypicalPositions string
}

// Availability tracks whether a band member is available on a date.
type Availability struct {
	Member    BandMember
	Available bool
	Reason    string // blockout reason if unavailable
}

// MusicMonthPlan holds music scheduling data for a single plan in a month.
type MusicMonthPlan struct {
	Date         string
	PlanID       string
	Title        string
	MusicLead    *TeamMember // nil if none assigned
	BandMembers  []TeamMember
	BlockedNames []string // "Name (reason)" strings
	BandSignup   *TeamSignup
}

// MusicMonth holds the full month overview.
type MusicMonth struct {
	YearMonth        string
	Plans            []MusicMonthPlan
	AppearanceCounts map[string]int // person name -> count
}

// TeamSignupAttrs holds normalized TeamSignup fields plus raw attributes.
type TeamSignupAttrs struct {
	SignupsEnabled *bool          `json:"signups_enabled,omitempty"`
	Raw            map[string]any `json:"raw,omitempty"`
}

// TeamSignup is a resolved team sign-up sheet for a plan.
type TeamSignup struct {
	ID       string          `json:"id"`
	PlanID   string          `json:"plan_id,omitempty"`
	TeamID   string          `json:"team_id,omitempty"`
	TeamName string          `json:"team_name,omitempty"`
	Attrs    TeamSignupAttrs `json:"attrs"`
}

// TeamSignupEnableResult describes an idempotent enable operation.
type TeamSignupEnableResult struct {
	PlanID     string     `json:"plan_id"`
	TeamSignup TeamSignup `json:"team_signup"`
	Created    bool       `json:"created"`
	Updated    bool       `json:"updated"`
}

// UnmarshalJSON keeps TeamSignup attrs tolerant as PCO evolves.
func (a *TeamSignupAttrs) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	a.Raw = raw
	if value, ok := raw["signups_enabled"].(bool); ok {
		a.SignupsEnabled = &value
	}
	return nil
}
