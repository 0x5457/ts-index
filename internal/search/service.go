package search

import (
	"context"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/models"
	"github.com/0x5457/ts-index/internal/storage"
)

// Service orchestrates semantic search for code snippets
type Service struct {
	Embedder embeddings.Embedder
	Vector   storage.VectorStore
}

// Search performs vector search and returns the top-k most similar code snippets
func (s *Service) Search(
	ctx context.Context,
	query string,
	topK int,
) ([]models.SemanticHit, error) {
	// Convert query to vector embedding
	qvec, err := s.Embedder.EmbedQuery(query)
	if err != nil {
		return nil, err
	}

	// Search for similar code snippets in the vector store
	hits, err := s.Vector.Query(qvec, topK)
	if err != nil {
		return nil, err
	}

	return hits, nil
}
