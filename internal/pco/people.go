package pco

import (
	"context"
	"encoding/json"

	"github.com/micahlee/pco-cli/internal/models"
)

// Me returns the current authenticated user's profile.
func (s *Service) Me(ctx context.Context) (*models.Person, error) {
	data, err := s.Client.Get(ctx, "/people/v2/me", nil)
	if err != nil {
		return nil, err
	}

	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}

	var attrs models.PersonAttrs
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return nil, err
	}

	return &models.Person{
		ID:    resource.ID,
		Attrs: attrs,
	}, nil
}
