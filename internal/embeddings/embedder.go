package embeddings

type Embedder interface {
	EmbedTexts(texts []string) ([][]float32, error)
	EmbedQuery(text string) ([]float32, error)
	ModelName() string
}
