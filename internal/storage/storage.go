package storage

import "github.com/0x5457/ts-index/internal/models"

type SymbolStore interface {
	UpsertSymbols(symbols []models.Symbol) error
	DeleteSymbolsByFile(file string) error
	FindByName(name string) ([]models.Symbol, error)
	GetByID(id string) (*models.Symbol, error)
}

type VectorStore interface {
	Upsert(chunks []models.CodeChunk, embeddings [][]float32) error
	DeleteByFile(file string) error
	Query(embedding []float32, topK int) ([]models.SemanticHit, error)
}
