package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/micahlee/pco-cli/internal/models"
)

// ListTeamMembers returns team member assignments for a plan.
func (s *Service) ListTeamMembers(ctx context.Context, planID string) ([]models.TeamMember, error) {
	data, err := s.Client.Get(ctx,
		s.servicePath()+"/plans/"+planID+"/team_members",
		url.Values{"per_page": {"50"}})
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	members := make([]models.TeamMember, len(resources))
	for i, r := range resources {
		var attrs models.TeamMemberAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}

		personID := ""
		if rels, ok := r.Relationships["person"]; ok {
			rid, _ := rels.One()
			if rid != nil {
				personID = rid.ID
			}
		}

		members[i] = models.TeamMember{
			ID:       r.ID,
			PersonID: personID,
			Attrs:    attrs,
		}
	}
	return members, nil
}

// SchedulePerson assigns a person to a plan position.
func (s *Service) SchedulePerson(ctx context.Context, planID, personID, teamID, position string) (string, error) {
	body := fmt.Sprintf(`{
		"data": {
			"type": "PlanPerson",
			"attributes": {
				"status": "U",
				"team_position_name": %q,
				"prepare_notification": true
			},
			"relationships": {
				"person": {"data": {"type": "Person", "id": %q}},
				"team": {"data": {"type": "Team", "id": %q}}
			}
		}
	}`, position, personID, teamID)

	data, err := s.Client.Post(ctx, s.servicePath()+"/plans/"+planID+"/team_members", body)
	if err != nil {
		return "", err
	}

	resource, err := models.ParseOne(data)
	if err != nil {
		return "", err
	}

	var attrs models.TeamMemberAttrs
	json.Unmarshal(resource.Attributes, &attrs)
	return attrs.Name, nil
}

// UnschedulePerson removes an assignment from a plan.
func (s *Service) UnschedulePerson(ctx context.Context, planID, assignID string) error {
	return s.Client.Delete(ctx, s.servicePath()+"/plans/"+planID+"/team_members/"+assignID)
}

// EnableSignups enables team sign-ups for a plan.
func (s *Service) EnableSignups(ctx context.Context, planID, teamID string) (string, error) {
	if teamID == "" {
		teamID = s.Config.BandTeamID
	}

	body := fmt.Sprintf(`{
		"data": {
			"type": "TeamSignup",
			"attributes": {"signups_enabled": true},
			"relationships": {
				"team": {"data": {"id": %s, "type": "Team"}}
			}
		}
	}`, teamID)

	data, err := s.Client.Post(ctx, s.servicePath()+"/plans/"+planID+"/team_signups", body)
	if err != nil {
		return "", err
	}

	resource, err := models.ParseOne(data)
	if err != nil {
		return "", err
	}

	return resource.ID, nil
}
