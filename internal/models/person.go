package models

// PersonAttrs holds the attributes of a PCO person.
type PersonAttrs struct {
	Name            string `json:"name"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	LoginIdentifier string `json:"login_identifier"`
}

// Person is a fully resolved person with ID.
type Person struct {
	ID    string
	Attrs PersonAttrs
}

// TeamMemberAttrs holds attributes for a plan team member assignment.
type TeamMemberAttrs struct {
	Name             string `json:"name"`
	TeamPositionName string `json:"team_position_name"`
	Status           string `json:"status"`
}

// TeamMember is a resolved team member assignment.
type TeamMember struct {
	ID       string
	PersonID string
	Attrs    TeamMemberAttrs
}
