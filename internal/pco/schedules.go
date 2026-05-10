package pco

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/micahlee/pco-cli/internal/models"
)

func (s *Service) personSchedulesPath() string {
	return "/services/v2/people/" + s.Config.PersonID + "/schedules"
}

// ListServeRequests returns pending (unconfirmed) serve requests.
func (s *Service) ListServeRequests(ctx context.Context) ([]models.Schedule, error) {
	data, err := s.Client.Get(ctx, s.personSchedulesPath(),
		url.Values{"per_page": {"50"}, "order": {"sort_date"}})
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	var pending []models.Schedule
	for _, r := range resources {
		var attrs models.ScheduleAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		if attrs.Status == "U" {
			pending = append(pending, models.Schedule{ID: r.ID, Attrs: attrs})
		}
	}
	return pending, nil
}

// AcceptServeRequest accepts a serve request.
func (s *Service) AcceptServeRequest(ctx context.Context, scheduleID string) error {
	_, err := s.Client.Post(ctx, s.personSchedulesPath()+"/"+scheduleID+"/accept", "{}")
	return err
}

// DeclineServeRequest declines a serve request.
func (s *Service) DeclineServeRequest(ctx context.Context, scheduleID string) error {
	_, err := s.Client.Post(ctx, s.personSchedulesPath()+"/"+scheduleID+"/decline", "{}")
	return err
}
