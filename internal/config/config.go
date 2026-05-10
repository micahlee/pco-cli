package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all PCO CLI configuration.
type Config struct {
	ClientID          string            `yaml:"client_id"`
	ClientSecret      string            `yaml:"client_secret"`
	PersonID          string            `yaml:"person_id"`
	ServiceTypeID     string            `yaml:"service_type_id"`
	BandTeamID        string            `yaml:"band_team_id"`
	ServiceRespTeamID string            `yaml:"service_resp_team_id"`
	DefaultTemplateID string            `yaml:"default_template_id"`
	TypicalPositions  map[string]string `yaml:"typical_positions"`
}

// defaults returns a Config with hardcoded default values.
func defaults() Config {
	return Config{
		PersonID:          "20101843",
		ServiceTypeID:     "643436",
		BandTeamID:        "2461416",
		ServiceRespTeamID: "2839232",
		DefaultTemplateID: "50925693",
		TypicalPositions: map[string]string{
			"20101843":  "Acoustic Guitar, Music Lead, Vocals",
			"30650677":  "Bass Guitar",
			"52462344":  "Percussion",
			"55768724":  "Keys, Vocals, Music Lead",
			"81615770":  "Keys, Vocals, Music Lead",
			"90183808":  "Acoustic Guitar, Vocals, Music Lead",
			"96703336":  "Vocals",
			"117793249": "Keys, Vocals",
		},
	}
}

// configPath returns the default config file path.
func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "pco", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pco", "config.yaml")
}

// Load reads configuration from the config file and environment variables.
// Environment variables override config file values.
func Load() (*Config, error) {
	cfg := defaults()

	// Read config file if it exists
	path := configPath()
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing config %s: %w", path, err)
		}
	}

	// Keychain credentials are used when the config file does not provide them.
	if cfg.ClientID == "" {
		if v, err := LoadCredentialFromKeychain(KeychainAccountClientID); err == nil {
			cfg.ClientID = v
		}
	}
	if cfg.ClientSecret == "" {
		if v, err := LoadCredentialFromKeychain(KeychainAccountClientSecret); err == nil {
			cfg.ClientSecret = v
		}
	}

	// Environment variables override config file and Keychain values.
	if v := os.Getenv("PCO_CLIENT_ID"); v != "" {
		cfg.ClientID = v
	}
	if v := os.Getenv("PCO_SECRET"); v != "" {
		cfg.ClientSecret = v
	}
	if v := os.Getenv("PCO_PERSON_ID"); v != "" {
		cfg.PersonID = v
	}
	if v := os.Getenv("PCO_SERVICE_TYPE_ID"); v != "" {
		cfg.ServiceTypeID = v
	}
	if v := os.Getenv("PCO_BAND_TEAM_ID"); v != "" {
		cfg.BandTeamID = v
	}
	if v := os.Getenv("PCO_SERVICE_RESP_TEAM_ID"); v != "" {
		cfg.ServiceRespTeamID = v
	}
	if v := os.Getenv("PCO_DEFAULT_TEMPLATE_ID"); v != "" {
		cfg.DefaultTemplateID = v
	}

	// Validate required credentials
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf(
			"PCO credentials not configured.\n\n"+
				"Run the interactive setup:\n"+
				"  pco init\n\n"+
				"Set via environment variables:\n"+
				"  export PCO_CLIENT_ID=your_client_id\n"+
				"  export PCO_SECRET=your_secret\n\n"+
				"Or create a config file at %s:\n"+
				"  client_id: your_client_id\n"+
				"  client_secret: your_secret", path)
	}

	return &cfg, nil
}
