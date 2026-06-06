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

// ListTeamSignups returns team sign-up sheets for a plan.
func (s *Service) ListTeamSignups(ctx context.Context, planID, teamID string) ([]models.TeamSignup, error) {
	if teamID == "" {
		teamID = s.Config.BandTeamID
	}

	data, err := s.Client.Get(ctx,
		s.servicePath()+"/plans/"+planID+"/team_signups",
		url.Values{"per_page": {"100"}, "include": {"team"}})
	if err != nil {
		return nil, err
	}

	resources, included, _, err := models.ParseListDocument(data)
	if err != nil {
		return nil, err
	}
	teams := includedResources(included, "Team")

	signups := make([]models.TeamSignup, 0, len(resources))
	for _, r := range resources {
		signup, err := teamSignupFromResource(planID, r, teams)
		if err != nil {
			return nil, err
		}
		if teamID != "" && signup.TeamID != teamID {
			continue
		}
		s.setDefaultTeamSignupName(&signup)
		signups = append(signups, signup)
	}
	return signups, nil
}

// EnableSignups enables team sign-ups for a plan without creating duplicates.
func (s *Service) EnableSignups(ctx context.Context, planID, teamID string) (*models.TeamSignupEnableResult, error) {
	if teamID == "" {
		teamID = s.Config.BandTeamID
	}

	signups, err := s.ListTeamSignups(ctx, planID, teamID)
	if err != nil {
		return nil, err
	}
	if len(signups) > 0 {
		if len(signups) > 1 {
			return nil, fmt.Errorf("found %d TeamSignup records for team %s on plan %s; clean up duplicates before enabling", len(signups), teamID, planID)
		}
		signup := signups[0]
		enabled := signup.Attrs.SignupsEnabled != nil && *signup.Attrs.SignupsEnabled
		if enabled {
			return &models.TeamSignupEnableResult{PlanID: planID, TeamSignup: signup}, nil
		}

		updated, err := s.setSignupEnabled(ctx, planID, signup.ID)
		if err != nil {
			return nil, err
		}
		updated.PlanID = planID
		updated.TeamID = signup.TeamID
		if updated.TeamName == "" {
			updated.TeamName = signup.TeamName
		}
		s.setDefaultTeamSignupName(updated)
		return &models.TeamSignupEnableResult{PlanID: planID, TeamSignup: *updated, Updated: true}, nil
	}

	body := fmt.Sprintf(`{
		"data": {
			"type": "TeamSignup",
			"attributes": {"signups_enabled": true},
			"relationships": {
				"team": {"data": {"id": %q, "type": "Team"}}
			}
		}
	}`, teamID)

	data, err := s.Client.Post(ctx, s.servicePath()+"/plans/"+planID+"/team_signups", body)
	if err != nil {
		return nil, err
	}

	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}

	signup, err := teamSignupFromResource(planID, *resource, nil)
	if err != nil {
		return nil, err
	}
	if signup.TeamID == "" {
		signup.TeamID = teamID
	}
	s.setDefaultTeamSignupName(&signup)
	return &models.TeamSignupEnableResult{PlanID: planID, TeamSignup: signup, Created: true}, nil
}

// EnableSignupsMonth enables team sign-ups for every plan in a month.
func (s *Service) EnableSignupsMonth(ctx context.Context, yearMonth, teamID string) ([]models.TeamSignupEnableResult, error) {
	plans, err := s.ListPlansForMonth(ctx, yearMonth)
	if err != nil {
		return nil, err
	}

	results := make([]models.TeamSignupEnableResult, 0, len(plans))
	for _, plan := range plans {
		result, err := s.EnableSignups(ctx, plan.ID, teamID)
		if err != nil {
			results = append(results, models.TeamSignupEnableResult{
				PlanID: plan.ID,
				Error:  err.Error(),
			})
			continue
		}
		results = append(results, *result)
	}
	return results, nil
}

// ListTeamSignupsMonth returns team sign-up sheets for every plan in a month.
func (s *Service) ListTeamSignupsMonth(ctx context.Context, yearMonth, teamID string) ([]models.TeamSignup, error) {
	plans, err := s.ListPlansForMonth(ctx, yearMonth)
	if err != nil {
		return nil, err
	}

	var signups []models.TeamSignup
	for _, plan := range plans {
		planSignups, err := s.ListTeamSignups(ctx, plan.ID, teamID)
		if err != nil {
			return nil, fmt.Errorf("plan %s: %w", plan.ID, err)
		}
		signups = append(signups, planSignups...)
	}
	return signups, nil
}

func (s *Service) setSignupEnabled(ctx context.Context, planID, signupID string) (*models.TeamSignup, error) {
	body := fmt.Sprintf(`{
		"data": {
			"type": "TeamSignup",
			"id": %q,
			"attributes": {"signups_enabled": true}
		}
	}`, signupID)

	data, err := s.Client.Patch(ctx, s.servicePath()+"/plans/"+planID+"/team_signups/"+signupID, body)
	if err != nil {
		return nil, err
	}
	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}
	signup, err := teamSignupFromResource(planID, *resource, nil)
	if err != nil {
		return nil, err
	}
	return &signup, nil
}

func (s *Service) setDefaultTeamSignupName(signup *models.TeamSignup) {
	if signup.TeamName == "" && signup.TeamID == s.Config.BandTeamID {
		signup.TeamName = "Band"
	}
}

func teamSignupFromResource(planID string, r models.Resource, teams map[string]models.Resource) (models.TeamSignup, error) {
	var attrs models.TeamSignupAttrs
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return models.TeamSignup{}, err
	}
	teamID := relationshipID(r, "team")
	signup := models.TeamSignup{
		ID:     r.ID,
		PlanID: planID,
		TeamID: teamID,
		Attrs:  attrs,
	}
	if team, ok := teams[teamID]; ok {
		signup.TeamName = resourceAttrString(team, "name")
	}
	return signup, nil
}

func includedResources(resources []models.Resource, resourceType string) map[string]models.Resource {
	result := make(map[string]models.Resource)
	for _, resource := range resources {
		if resource.Type == resourceType {
			result[resource.ID] = resource
		}
	}
	return result
}
