package fx

import (
	"github.com/0x5457/ts-index/internal/embeddings"
	"go.uber.org/fx"
)

// EmbeddingsParams represents dependencies for embeddings components
type EmbeddingsParams struct {
	fx.In

	Config *Config
}

// NewEmbedder creates a new embedder instance
func NewEmbedder(params EmbeddingsParams) embeddings.Embedder {
	return embeddings.NewApi(params.Config.EmbedURL)
}

// NewLocalEmbedder creates a local embedder for testing
func NewLocalEmbedder(dimension int) embeddings.Embedder {
	return embeddings.NewLocal(dimension)
}

// EmbeddingsModule provides embeddings components
var EmbeddingsModule = fx.Module("embeddings",
	fx.Provide(NewEmbedder),
)
