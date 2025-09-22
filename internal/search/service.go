package search

import (
	"context"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/featurizer"
	"github.com/0x5457/ts-index/internal/models"
	"github.com/0x5457/ts-index/internal/storage"
)

// Service orchestrates semantic search with feature extraction and returns enriched hits.
type Service struct {
	Embedder   embeddings.Embedder
	Vector     storage.VectorStore
	Featurizer *featurizer.Featurizer
	LLMConfig  featurizer.LLMConfig
}

type EnrichedHit struct {
	Hit      models.SemanticHit
	Features map[string]float64 // feature_id -> coefficient (0..1)
}

// SearchWithFeatures performs vector search and, for the query text, extracts features
// via the featurizer, returning both similarity hits and feature coefficients.
func (s *Service) SearchWithFeatures(
	ctx context.Context,
	query string,
	topK int,
	samples int,
	temperature float64,
) ([]EnrichedHit, featurizer.FeatureEmbedding, error) {
	// 1) semantic embedding + search
	qvec, err := s.Embedder.EmbedQuery(query)
	if err != nil {
		return nil, featurizer.FeatureEmbedding{}, err
	}
	hits, err := s.Vector.Query(qvec, topK)
	if err != nil {
		return nil, featurizer.FeatureEmbedding{}, err
	}

	// 2) feature extraction for the query
	emb, err := s.Featurizer.Embed(ctx, query, s.LLMConfig, temperature, samples)
	if err != nil {
		return nil, featurizer.FeatureEmbedding{}, err
	}

	// Build feature coefficients map once
	coeffs := map[string]float64{}
	for _, ft := range s.Featurizer.Features {
		if v, ok := emb.Coefficient(ft.Identifier); ok {
			coeffs[ft.Identifier] = v
		}
	}

	enriched := make([]EnrichedHit, len(hits))
	for i, h := range hits {
		enriched[i] = EnrichedHit{Hit: h, Features: coeffs}
	}
	return enriched, emb, nil
}
