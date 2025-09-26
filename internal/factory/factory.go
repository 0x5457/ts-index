package factory

import (
	"fmt"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/parser"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/0x5457/ts-index/internal/storage"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
)

// ComponentConfig holds configuration for creating components
type ComponentConfig struct {
	DBPath          string
	EmbedURL        string
	VectorDimension int
}

// Components holds all the main components
type Components struct {
	Parser   parser.Parser
	Embedder embeddings.Embedder
	SymStore storage.SymbolStore
	VecStore storage.VectorStore
	Searcher *search.Service
}

// ComponentFactory creates and manages component instances
type ComponentFactory struct {
	config ComponentConfig
}

// NewComponentFactory creates a new component factory
func NewComponentFactory(config ComponentConfig) *ComponentFactory {
	// Set defaults
	if config.EmbedURL == "" {
		config.EmbedURL = "http://localhost:8000/embed"
	}
	if config.VectorDimension == 0 {
		config.VectorDimension = 0 // Will be inferred
	}

	return &ComponentFactory{config: config}
}

// CreateComponents creates all components with the given configuration
func (f *ComponentFactory) CreateComponents() (*Components, error) {
	if f.config.DBPath == "" {
		return nil, fmt.Errorf("database path must be specified")
	}

	// Create parser
	parser := f.CreateParser()

	// Create embedder
	embedder := f.CreateEmbedder()

	// Create symbol store
	symStore, err := f.CreateSymbolStore()
	if err != nil {
		return nil, fmt.Errorf("create symbol store failed: %w", err)
	}

	// Create vector store
	vecStore, err := f.CreateVectorStore()
	if err != nil {
		return nil, fmt.Errorf("create vector store failed: %w", err)
	}

	// Create search service
	searcher := f.CreateSearchService(embedder, vecStore)

	return &Components{
		Parser:   parser,
		Embedder: embedder,
		SymStore: symStore,
		VecStore: vecStore,
		Searcher: searcher,
	}, nil
}

// CreateParser creates a parser instance
func (f *ComponentFactory) CreateParser() parser.Parser {
	return tsparser.New()
}

// CreateEmbedder creates an embedder instance
func (f *ComponentFactory) CreateEmbedder() embeddings.Embedder {
	return embeddings.NewApi(f.config.EmbedURL)
}

// CreateLocalEmbedder creates a local embedder for testing
func (f *ComponentFactory) CreateLocalEmbedder(dimension int) embeddings.Embedder {
	return embeddings.NewLocal(dimension)
}

// CreateSymbolStore creates a symbol store instance
func (f *ComponentFactory) CreateSymbolStore() (storage.SymbolStore, error) {
	return sqlite.New(f.config.DBPath)
}

// CreateVectorStore creates a vector store instance
func (f *ComponentFactory) CreateVectorStore() (storage.VectorStore, error) {
	return sqlvec.New(f.config.DBPath, f.config.VectorDimension)
}

// CreateSearchService creates a search service instance
func (f *ComponentFactory) CreateSearchService(
	embedder embeddings.Embedder,
	vecStore storage.VectorStore,
) *search.Service {
	return &search.Service{
		Embedder: embedder,
		Vector:   vecStore,
	}
}

// CreateIndexer creates an indexer instance with the given components
func (f *ComponentFactory) CreateIndexer(components *Components) *pipeline.Indexer {
	return pipeline.New(
		components.Parser,
		components.Embedder,
		components.SymStore,
		components.VecStore,
		pipeline.Options{},
	)
}

// CreateIndexerWithOptions creates an indexer with custom options
func (f *ComponentFactory) CreateIndexerWithOptions(
	components *Components,
	opts pipeline.Options,
) *pipeline.Indexer {
	return pipeline.New(
		components.Parser,
		components.Embedder,
		components.SymStore,
		components.VecStore,
		opts,
	)
}

// Cleanup releases resources held by components
func (c *Components) Cleanup() error {
	if c.VecStore != nil {
		if closer, ok := c.VecStore.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				return fmt.Errorf("close vector store failed: %w", err)
			}
		}
	}
	return nil
}
