package indexer

import "github.com/0x5457/ts-index/internal/models"

type Indexer interface {
	IndexProject(path string) error
	IndexFile(path string) error
	SearchSymbol(name string) ([]models.SymbolHit, error)
	SearchSemantic(query string, topK int) ([]models.SemanticHit, error)
}
