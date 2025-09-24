package pipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
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

	p := tsparser.New()
	e := embeddings.NewLocal(8)
	s, err := sqlite.New(db)
	if err != nil {
		t.Fatal(err)
	}
	v, err := sqlvec.New(db, 0)
	if err != nil {
		t.Fatal(err)
	}
	idx := pipeline.New(p, e, s, v, pipeline.Options{EmbedBatchSize: 2})

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
