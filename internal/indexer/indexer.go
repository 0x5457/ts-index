package indexer

import (
	"context"

	"github.com/0x5457/ts-index/internal/models"
)

type Indexer interface {
	IndexProject(path string) error
	IndexFile(path string) error
	IndexFileWithRoot(root, path string) error
	SearchSymbol(name string) ([]models.SymbolHit, error)
	SearchSemantic(query string, topK int) ([]models.SemanticHit, error)

	IndexProjectProgress(
		ctx context.Context,
		path string,
	) (<-chan models.IndexProgress, <-chan error)
}
