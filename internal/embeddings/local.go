package embeddings

import (
	"crypto/sha1"
)

type LocalEmbedder struct {
	dim int
}

func NewLocal(dim int) *LocalEmbedder { return &LocalEmbedder{dim: dim} }

func (e *LocalEmbedder) ModelName() string { return "local-fixed" }

func (e *LocalEmbedder) EmbedTexts(texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i, t := range texts {
		vecs[i] = hashToVector(t, e.dim)
	}
	return vecs, nil
}

func (e *LocalEmbedder) EmbedQuery(text string) ([]float32, error) {
	return hashToVector(text, e.dim), nil
}

func hashToVector(s string, dim int) []float32 {
	h := sha1.Sum([]byte(s))
	vec := make([]float32, dim)
	for i := 0; i < dim; i++ {
		// repeat hash bytes to fill dim and normalize roughly
		b := h[i%len(h)]
		vec[i] = (float32(int8(b)) / 127.0)
	}
	return vec
}
