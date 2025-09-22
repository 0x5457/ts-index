package memory

import (
	"math"
	"sync"

	"github.com/0x5457/ts-index/internal/models"
)

type item struct {
	chunk models.CodeChunk
	vec   []float32
}

type InMemoryVectorStore struct {
	mu   sync.RWMutex
	data map[string][]item // file -> items
}

func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{data: make(map[string][]item)}
}

func (s *InMemoryVectorStore) Upsert(chunks []models.CodeChunk, embeddings [][]float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// group by file
	tmp := make(map[string][]item)
	for i, ch := range chunks {
		it := item{chunk: ch, vec: embeddings[i]}
		tmp[ch.File] = append(tmp[ch.File], it)
	}
	for file, items := range tmp {
		// remove existing entries for the same IDs
		existing := s.data[file]
		idToNew := make(map[string]item)
		for _, it := range items {
			idToNew[it.chunk.ID] = it
		}
		var merged []item
		for _, it := range existing {
			if _, ok := idToNew[it.chunk.ID]; !ok {
				merged = append(merged, it)
			}
		}
		merged = append(merged, items...)
		s.data[file] = merged
	}
	return nil
}

func (s *InMemoryVectorStore) DeleteByFile(file string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, file)
	return nil
}

func (s *InMemoryVectorStore) Query(embedding []float32, topK int) ([]models.SemanticHit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var all []item
	for _, items := range s.data {
		all = append(all, items...)
	}
	// compute cosine similarity
	type scored struct {
		it    item
		score float32
	}
	scoredList := make([]scored, 0, len(all))
	for _, it := range all {
		scoredList = append(scoredList, scored{it: it, score: cosine(it.vec, embedding)})
	}
	// partial sort: simple selection for small sizes
	if topK > len(scoredList) {
		topK = len(scoredList)
	}
	for i := 0; i < topK; i++ {
		best := i
		for j := i + 1; j < len(scoredList); j++ {
			if scoredList[j].score > scoredList[best].score {
				best = j
			}
		}
		scoredList[i], scoredList[best] = scoredList[best], scoredList[i]
	}
	var hits []models.SemanticHit
	for i := 0; i < topK; i++ {
		hits = append(
			hits,
			models.SemanticHit{Chunk: scoredList[i].it.chunk, Score: scoredList[i].score},
		)
	}
	return hits, nil
}

func cosine(a, b []float32) float32 {
	var dot float64
	var na float64
	var nb float64
	for i := 0; i < len(a) && i < len(b); i++ {
		dot += float64(a[i] * b[i])
		na += float64(a[i] * a[i])
		nb += float64(b[i] * b[i])
	}
	den := math.Sqrt(na) * math.Sqrt(nb)
	if den == 0 {
		return 0
	}
	return float32(dot / den)
}
