package models

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
}

// MusicMonth holds the full month overview.
type MusicMonth struct {
	YearMonth        string
	Plans            []MusicMonthPlan
	AppearanceCounts map[string]int // person name -> count
}
