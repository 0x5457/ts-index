package configfx

import (
	"github.com/0x5457/ts-index/internal/constants"
	"go.uber.org/fx"
)

// Config holds the application configuration
type Config struct {
	DBPath          string
	EmbedURL        string
	VectorDimension int
	Project         string // Optional project path for pre-indexing
}

// Params represents the parameters needed to create configuration
type Params struct {
	fx.In

	DBPath   string `name:"dbPath"   optional:"true"`
	EmbedURL string `name:"embedURL" optional:"true"`
	Project  string `name:"project"  optional:"true"`
}

// NewConfig creates a new configuration with defaults
func NewConfig(params Params) *Config {
	config := &Config{
		DBPath:          params.DBPath,
		EmbedURL:        params.EmbedURL,
		VectorDimension: 0, // Will be inferred
		Project:         params.Project,
	}

	// Set defaults
	if config.EmbedURL == "" {
		config.EmbedURL = constants.DefaultEmbedURL
	}

	return config
}

// Module provides configuration for the application
var Module = fx.Module("config",
	fx.Provide(NewConfig),
)
