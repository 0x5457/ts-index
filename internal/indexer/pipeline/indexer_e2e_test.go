package pipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0x5457/ts-index/internal/factory"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
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

	// Create component factory
	componentFactory := factory.NewComponentFactory(factory.ComponentConfig{
		DBPath: db,
	})

	// Create components (but use local embedder for testing)
	components, err := componentFactory.CreateComponents()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cleanupErr := components.Cleanup(); cleanupErr != nil {
			t.Logf("cleanup error: %v", cleanupErr)
		}
	}()

	// Override with local embedder for testing
	components.Embedder = componentFactory.CreateLocalEmbedder(8)

	idx := componentFactory.CreateIndexerWithOptions(
		components,
		pipeline.Options{EmbedBatchSize: 2},
	)

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
