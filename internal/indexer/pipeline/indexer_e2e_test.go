package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0x5457/ts-index/internal/config/configfx"
	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer"
	"github.com/0x5457/ts-index/internal/indexer/indexerfx"
	"github.com/0x5457/ts-index/internal/parser/parserfx"
	"github.com/0x5457/ts-index/internal/storage/storagefx"
	"go.uber.org/fx"
)

func Test_Indexer_E2E_TS(t *testing.T) {
	tmp := t.TempDir()
	// prepare small TS project
	src := `export function add(a:number,b:number){return a+b}`
	path := filepath.Join(tmp, "a.ts")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	db := filepath.Join(tmp, "index.db")

	// Create test local embedder
	testEmbedderModule := fx.Module("test-embeddings",
		fx.Provide(func() embeddings.Embedder {
			return embeddings.NewLocal(8) // 8-dimensional for testing
		}),
		fx.Decorate(func(embedder embeddings.Embedder) embeddings.Embedder {
			return embedder // Override the API embedder with local one
		}),
	)

	// Create Fx app with test configuration
	var idx indexer.Indexer
	app := fx.New(
		configfx.Module,
		parserfx.Module,
		testEmbedderModule, // Use test embedder instead of embeddingsfx.Module
		storagefx.Module,
		indexerfx.Module,
		fx.Supply(
			fx.Annotate(db, fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate("", fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate("", fx.ResultTags(`name:"project"`)),
		),
		fx.Populate(&idx),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := app.Stop(ctx); err != nil {
			t.Logf("cleanup error: %v", err)
		}
	}()

	if err := idx.IndexProject(tmp); err != nil {
		t.Fatalf("index project: %v", err)
	}

	// symbol search
	syms, err := idx.SearchSymbol("add")
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) == 0 {
		t.Fatalf("expected symbol 'add'")
	}

	// semantic search
	hits, err := idx.SearchSemantic("addition function", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected hits")
	}
}
