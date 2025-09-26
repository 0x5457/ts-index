package embeddingsfx

import (
	"github.com/0x5457/ts-index/internal/config/configfx"
	"github.com/0x5457/ts-index/internal/embeddings"
	"go.uber.org/fx"
)

// Params represents dependencies for embeddings components
type Params struct {
	fx.In

	Config *configfx.Config
}

// NewEmbedder creates a new embedder instance
func NewEmbedder(params Params) embeddings.Embedder {
	return embeddings.NewApi(params.Config.EmbedURL)
}

// NewLocalEmbedder creates a local embedder for testing
func NewLocalEmbedder(dimension int) embeddings.Embedder {
	return embeddings.NewLocal(dimension)
}

// Module provides embeddings components
var Module = fx.Module("embeddings",
	fx.Provide(NewEmbedder),
)
