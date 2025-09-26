package appfx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0x5457/ts-index/cmd/cmdsfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestAppModule(t *testing.T) {
	// Test that all modules can be loaded together
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var runner *cmdsfx.CommandRunner

	app := fx.New(
		Module,
		fx.Supply(
			fx.Annotate(dbPath, fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate("http://localhost:8000/embed", fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate("", fx.ResultTags(`name:"project"`)),
		),
		fx.Populate(&runner),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
		_ = os.Remove(dbPath) // cleanup, ignore error
	}()

	assert.NotNil(t, runner)
}

func TestNewAppWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	app := NewAppWithConfig(dbPath, "http://localhost:8000/embed", "")

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
		_ = os.Remove(dbPath) // cleanup, ignore error
	}()
}
