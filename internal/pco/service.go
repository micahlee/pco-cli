package pco

import (
	"github.com/micahlee/pco-cli/internal/api"
	"github.com/micahlee/pco-cli/internal/config"
)

// Service provides high-level operations against the PCO API.
type Service struct {
	Client *api.Client
	Config *config.Config
}

// NewService creates a Service from the given config.
func NewService(cfg *config.Config) *Service {
	client := api.New(cfg.ClientID, cfg.ClientSecret)
	return &Service{
		Client: client,
		Config: cfg,
	}
}

// servicePath returns the base path for the configured service type.
func (s *Service) servicePath() string {
	return "/services/v2/service_types/" + s.Config.ServiceTypeID
}
