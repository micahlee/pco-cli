package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/micahlee/pco-cli/internal/models"
)

func (s *Service) personBlockoutsPath(personID string) string {
	return "/services/v2/people/" + personID + "/blockouts"
}

// ListBlockouts returns blockout dates for the configured user.
func (s *Service) ListBlockouts(ctx context.Context) ([]models.Blockout, error) {
	return s.ListBlockoutsForPerson(ctx, s.Config.PersonID)
}

// ListBlockoutsForPerson returns blockout dates for a specific person.
func (s *Service) ListBlockoutsForPerson(ctx context.Context, personID string) ([]models.Blockout, error) {
	resources, err := s.Client.GetAll(ctx, s.personBlockoutsPath(personID),
		url.Values{"per_page": {"100"}, "order": {"starts_at"}})
	if err != nil {
		return nil, err
	}

	blockouts := make([]models.Blockout, len(resources))
	for i, r := range resources {
		var attrs models.BlockoutAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		blockouts[i] = models.Blockout{ID: r.ID, Attrs: attrs}
	}
	return blockouts, nil
}

// AddBlockout creates a new blockout date for the configured user.
func (s *Service) AddBlockout(ctx context.Context, start, end, reason string) (*models.Blockout, error) {
	body := fmt.Sprintf(`{
		"data": {
			"type": "Blockout",
			"attributes": {
				"starts_at": "%sT00:00:00Z",
				"ends_at": "%sT23:59:59Z",
				"reason": %q,
				"repeat_frequency": "no_repeat"
			}
		}
	}`, start, end, reason)

	data, err := s.Client.Post(ctx, s.personBlockoutsPath(s.Config.PersonID), body)
	if err != nil {
		return nil, err
	}

	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}

	var attrs models.BlockoutAttrs
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return nil, err
	}

	return &models.Blockout{ID: resource.ID, Attrs: attrs}, nil
}

// DeleteBlockout removes a blockout date.
func (s *Service) DeleteBlockout(ctx context.Context, blockoutID string) error {
	return s.Client.Delete(ctx, s.personBlockoutsPath(s.Config.PersonID)+"/"+blockoutID)
}
