package searchfx

import (
	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/0x5457/ts-index/internal/storage"
	"go.uber.org/fx"
)

// Params represents dependencies for search service
type Params struct {
	fx.In

	Embedder embeddings.Embedder
	VecStore storage.VectorStore
}

// NewSearchService creates a new search service instance
func NewSearchService(params Params) *search.Service {
	return &search.Service{
		Embedder: params.Embedder,
		Vector:   params.VecStore,
	}
}

// Module provides search components
var Module = fx.Module("search",
	fx.Provide(NewSearchService),
)
