package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/micahlee/pco-cli/internal/models"
)

// ListPlans returns upcoming or past plans.
func (s *Service) ListPlans(ctx context.Context, filter string, count int) ([]models.Plan, error) {
	params := url.Values{
		"filter":   {filter},
		"per_page": {strconv.Itoa(count)},
		"order":    {"sort_date"},
	}
	if filter == "past" {
		params.Set("order", "-sort_date")
	}

	data, err := s.Client.Get(ctx, s.servicePath()+"/plans", params)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	plans := make([]models.Plan, len(resources))
	for i, r := range resources {
		var attrs models.PlanAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		plans[i] = models.Plan{ID: r.ID, Attrs: attrs}
	}
	return plans, nil
}

// GetPlan returns a single plan by ID.
func (s *Service) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	data, err := s.Client.Get(ctx, s.servicePath()+"/plans/"+planID, nil)
	if err != nil {
		return nil, err
	}

	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}

	var attrs models.PlanAttrs
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return nil, err
	}

	return &models.Plan{ID: resource.ID, Attrs: attrs}, nil
}

// ListPlanItems returns all items for a plan.
func (s *Service) ListPlanItems(ctx context.Context, planID string) ([]models.PlanItem, error) {
	return s.listPlanItems(ctx, planID, nil)
}

// ListPlanSongs returns only song items for a plan.
func (s *Service) ListPlanSongs(ctx context.Context, planID string) ([]models.PlanItem, error) {
	params := url.Values{"filter": {"songs"}}
	return s.listPlanItems(ctx, planID, params)
}

func (s *Service) listPlanItems(ctx context.Context, planID string, extra url.Values) ([]models.PlanItem, error) {
	params := url.Values{"per_page": {"50"}}
	for k, v := range extra {
		params[k] = v
	}

	data, err := s.Client.Get(ctx, s.servicePath()+"/plans/"+planID+"/items", params)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	items := make([]models.PlanItem, len(resources))
	for i, r := range resources {
		var attrs models.PlanItemAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}

		songID := ""
		if rel, ok := r.Relationships["song"]; ok {
			rid, _ := rel.One()
			if rid != nil {
				songID = rid.ID
			}
		}

		items[i] = models.PlanItem{ID: r.ID, SongID: songID, Attrs: attrs}
	}
	return items, nil
}

// ListTemplates returns all plan templates.
func (s *Service) ListTemplates(ctx context.Context) ([]models.PlanTemplate, error) {
	data, err := s.Client.Get(ctx, s.servicePath()+"/plan_templates", nil)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	templates := make([]models.PlanTemplate, len(resources))
	for i, r := range resources {
		var attrs models.PlanTemplateAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		templates[i] = models.PlanTemplate{ID: r.ID, Attrs: attrs}
	}
	return templates, nil
}

// CreatePlan creates a new plan from a template on the given date.
func (s *Service) CreatePlan(ctx context.Context, date string, templateID string) ([]models.Plan, error) {
	if templateID == "" {
		templateID = s.Config.DefaultTemplateID
	}

	body := fmt.Sprintf(`{
		"data": {
			"type": "Plan",
			"attributes": {
				"base_date": %q,
				"copy_items": true,
				"copy_people": true,
				"copy_notes": true,
				"count": 1
			},
			"relationships": {
				"template": {
					"data": [{"type": "PlanTemplate", "id": %q}]
				}
			}
		}
	}`, date, templateID)

	data, err := s.Client.Post(ctx, s.servicePath()+"/create_plans", body)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	plans := make([]models.Plan, len(resources))
	for i, r := range resources {
		var attrs models.PlanAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		plans[i] = models.Plan{ID: r.ID, Attrs: attrs}
	}
	return plans, nil
}
