package fx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/parser"
	"github.com/0x5457/ts-index/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestConfigModule(t *testing.T) {
	var config *Config
	app := fx.New(
		ConfigModule,
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

func TestParserModule(t *testing.T) {
	var parser parser.Parser
	app := fx.New(
		ParserModule,
		fx.Populate(&parser),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
	}()

	assert.NotNil(t, parser)
}

func TestEmbeddingsModule(t *testing.T) {
	var embedder embeddings.Embedder
	app := fx.New(
		ConfigModule,
		EmbeddingsModule,
		fx.Supply(
			fx.Annotate("", fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate("http://localhost:8000/embed", fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate("", fx.ResultTags(`name:"project"`)),
		),
		fx.Populate(&embedder),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
	}()

	assert.NotNil(t, embedder)
	assert.Equal(t, "api", embedder.ModelName())
}

func TestStorageModule(t *testing.T) {
	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var symbolStore storage.SymbolStore
	var vectorStore storage.VectorStore

	app := fx.New(
		ConfigModule,
		StorageModule,
		fx.Supply(
			fx.Annotate(dbPath, fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate("http://localhost:8000/embed", fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate("", fx.ResultTags(`name:"project"`)),
		),
		fx.Populate(&symbolStore, &vectorStore),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
		_ = os.Remove(dbPath) // cleanup, ignore error
	}()

	assert.NotNil(t, symbolStore)
	assert.NotNil(t, vectorStore)
}

func TestAppModule(t *testing.T) {
	// Test that all modules can be loaded together
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var runner *CommandRunner

	app := fx.New(
		AppModule,
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
	assert.NotNil(t, runner.config)
	assert.NotNil(t, runner.searchService)
	assert.NotNil(t, runner.indexer)
	assert.NotNil(t, runner.mcpServer)
}
