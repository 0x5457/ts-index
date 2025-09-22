package pipeline

import (
	"context"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/models"
	"github.com/0x5457/ts-index/internal/parser"
	"github.com/0x5457/ts-index/internal/storage"
)

type Options struct {
	ParseWorkers   int
	EmbedBatchSize int
	EmbedWorkers   int
}

type Indexer struct {
	p   parser.Parser
	e   embeddings.Embedder
	sym storage.SymbolStore
	vec storage.VectorStore
	opt Options
}

func New(
	p parser.Parser,
	e embeddings.Embedder,
	s storage.SymbolStore,
	v storage.VectorStore,
	opt Options,
) *Indexer {
	if opt.ParseWorkers <= 0 {
		opt.ParseWorkers = runtime.NumCPU()
	}
	if opt.EmbedWorkers <= 0 {
		opt.EmbedWorkers = runtime.NumCPU()
	}
	if opt.EmbedBatchSize <= 0 {
		opt.EmbedBatchSize = 64
	}
	return &Indexer{p: p, e: e, sym: s, vec: v, opt: opt}
}

func (i *Indexer) IndexProject(root string) error {
	files, err := listTSFiles(root)
	if err != nil {
		return err
	}

	// Stage 1: parse files concurrently
	parseCh := make(chan string, len(files))
	resCh := make(chan struct {
		syms []models.Symbol
		chs  []models.CodeChunk
		err  error
	}, len(files))
	var wgParse sync.WaitGroup
	for w := 0; w < i.opt.ParseWorkers; w++ {
		wgParse.Add(1)
		go func() {
			defer wgParse.Done()
			for f := range parseCh {
				syms, chs, err := i.p.ParseFile(f)
				resCh <- struct {
					syms []models.Symbol
					chs  []models.CodeChunk
					err  error
				}{syms, chs, err}
			}
		}()
	}
	for _, f := range files {
		parseCh <- f
	}
	close(parseCh)
	go func() { wgParse.Wait(); close(resCh) }()

	// Stage 2: collect and embed in batches
	var allSyms []models.Symbol
	var batchChs []models.CodeChunk
	flush := func(chs []models.CodeChunk) error {
		if len(chs) == 0 {
			return nil
		}
		texts := make([]string, len(chs))
		for idx, ch := range chs {
			texts[idx] = buildEmbedText(ch)
		}
		vecs, err := i.e.EmbedTexts(texts)
		if err != nil {
			return err
		}
		return i.vec.Upsert(chs, vecs)
	}
	for r := range resCh {
		if r.err != nil {
			return r.err
		}
		allSyms = append(allSyms, r.syms...)
		batchChs = append(batchChs, r.chs...)
		for len(batchChs) >= i.opt.EmbedBatchSize {
			if err := flush(batchChs[:i.opt.EmbedBatchSize]); err != nil {
				return err
			}
			batchChs = batchChs[i.opt.EmbedBatchSize:]
		}
	}
	if err := flush(batchChs); err != nil {
		return err
	}

	// upsert symbols once at the end (can be large; consider chunking if needed)
	if err := i.sym.UpsertSymbols(allSyms); err != nil {
		return err
	}
	return nil
}

func (i *Indexer) IndexFile(path string) error {
	if err := i.sym.DeleteSymbolsByFile(path); err != nil {
		return err
	}
	if err := i.vec.DeleteByFile(path); err != nil {
		return err
	}
	syms, chs, err := i.p.ParseFile(path)
	if err != nil {
		return err
	}
	texts := make([]string, len(chs))
	for idx, ch := range chs {
		texts[idx] = buildEmbedText(ch)
	}
	vecs, err := i.e.EmbedTexts(texts)
	if err != nil {
		return err
	}
	if err := i.sym.UpsertSymbols(syms); err != nil {
		return err
	}
	return i.vec.Upsert(chs, vecs)
}

func (i *Indexer) SearchSymbol(name string) ([]models.SymbolHit, error) {
	syms, err := i.sym.FindByName(name)
	if err != nil {
		return nil, err
	}
	res := make([]models.SymbolHit, len(syms))
	for idx := range syms {
		res[idx] = models.SymbolHit{Symbol: syms[idx]}
	}
	return res, nil
}

func (i *Indexer) SearchSemantic(query string, topK int) ([]models.SemanticHit, error) {
	vec, err := i.e.EmbedQuery(query)
	if err != nil {
		return nil, err
	}
	return i.vec.Query(vec, topK)
}

func listTSFiles(root string) ([]string, error) {
	var files []string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		if (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) &&
			!strings.HasSuffix(path, ".d.ts") {
			files = append(files, path)
		}
		return nil
	})
	return files, walkErr
}

func buildEmbedText(ch models.CodeChunk) string {
	var b strings.Builder
	b.WriteString(ch.Signature)
	b.WriteString("\n")
	if ch.Docstring != "" {
		b.WriteString(ch.Docstring)
		b.WriteString("\n")
	}
	b.WriteString(ch.Content)
	return b.String()
}

var _ = context.Background
