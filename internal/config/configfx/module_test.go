package configfx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestConfigModule(t *testing.T) {
	var config *Config
	app := fx.New(
		Module,
		fx.Supply(
			fx.Annotate("/tmp/test.db", fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate("http://localhost:8000/embed", fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate("", fx.ResultTags(`name:"project"`)),
		),
		fx.Populate(&config),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
	}()

	assert.NotNil(t, config)
	assert.Equal(t, "/tmp/test.db", config.DBPath)
	assert.Equal(t, "http://localhost:8000/embed", config.EmbedURL)
	assert.Equal(t, "", config.Project)
}

func TestConfigDefaults(t *testing.T) {
	var config *Config
	app := fx.New(
		Module,
		fx.Supply(
			fx.Annotate("", fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate("", fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate("", fx.ResultTags(`name:"project"`)),
		),
		fx.Populate(&config),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
	}()

	assert.NotNil(t, config)
	assert.Equal(t, "http://localhost:8000/embed", config.EmbedURL) // Default value
}
