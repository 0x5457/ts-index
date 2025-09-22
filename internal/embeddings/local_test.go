package embeddings_test

import (
	"testing"

	"github.com/0x5457/ts-index/internal/embeddings"
)

func Test_LocalEmbedder_Deterministic(t *testing.T) {
	e := embeddings.NewLocal(8)
	v1, _ := e.EmbedQuery("hello")
	v2, _ := e.EmbedQuery("hello")
	if len(v1) != 8 || len(v2) != 8 {
		t.Fatalf("unexpected dim")
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("vectors differ at %d", i)
		}
	}
}
