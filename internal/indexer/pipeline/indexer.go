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
	progCh, errCh := i.IndexProjectProgress(context.Background(), root)
	var retErr error
	for range progCh {
		// consume progress silently in sync mode
	}
	if errCh != nil {
		for err := range errCh {
			if err != nil {
				retErr = err
			}
		}
	}
	return retErr
}

func (i *Indexer) IndexProjectProgress(
	ctx context.Context,
	root string,
) (<-chan models.IndexProgress, <-chan error) {
	progCh := make(chan models.IndexProgress, 128)
	errCh := make(chan error, 1)

	send := func(p models.IndexProgress) {
		select {
		case <-ctx.Done():
			return
		case progCh <- p:
		}
	}

	go func() {
		defer close(progCh)
		defer close(errCh)

		files, err := listTSFiles(root)
		if err != nil {
			errCh <- err
			return
		}
		totalFiles := len(files)
		send(models.IndexProgress{
			Stage:      models.IndexStageScan,
			TotalFiles: totalFiles,
			Message:    "scan complete",
			Percent:    0,
		})

		// Stage 1: parse files concurrently
		parseCh := make(chan string, totalFiles)
		type parseRes struct {
			syms []models.Symbol
			chs  []models.CodeChunk
			err  error
			file string
		}
		resCh := make(chan parseRes, totalFiles)

		var wgParse sync.WaitGroup
		for w := 0; w < i.opt.ParseWorkers; w++ {
			wgParse.Add(1)
			go func() {
				defer wgParse.Done()
				for f := range parseCh {
					syms, chs, err := i.p.ParseFile(f)
					select {
					case <-ctx.Done():
						return
					case resCh <- parseRes{syms: syms, chs: chs, err: err, file: f}:
					}
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
		parsedFiles := 0
		totalChunks := 0
		embeddedChunks := 0

		// Percent policy:
		// - Parse 60%
		// - Embed 35%
		// - Symbol upsert 5%
		updateParseProgress := func(currentFile string) {
			pct := float32(0)
			if totalFiles > 0 {
				pct = 0.6 * float32(parsedFiles) / float32(totalFiles)
			}
			send(models.IndexProgress{
				Stage:          models.IndexStageParse,
				TotalFiles:     totalFiles,
				ParsedFiles:    parsedFiles,
				TotalChunks:    totalChunks,
				EmbeddedChunks: embeddedChunks,
				CurrentFile:    currentFile,
				Percent:        pct,
			})
		}
		updateEmbedProgress := func() {
			pct := float32(0.6)
			if totalChunks > 0 {
				pct = 0.6 + 0.35*float32(embeddedChunks)/float32(totalChunks)
			}
			send(models.IndexProgress{
				Stage:          models.IndexStageEmbed,
				TotalFiles:     totalFiles,
				ParsedFiles:    parsedFiles,
				TotalChunks:    totalChunks,
				EmbeddedChunks: embeddedChunks,
				Percent:        pct,
			})
		}

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
			if err := i.vec.Upsert(chs, vecs); err != nil {
				return err
			}
			embeddedChunks += len(chs)
			updateEmbedProgress()
			return nil
		}

		for r := range resCh {
			if r.err != nil {
				errCh <- r.err
				return
			}
			allSyms = append(allSyms, r.syms...)
			batchChs = append(batchChs, r.chs...)
			totalChunks += len(r.chs)
			parsedFiles++
			updateParseProgress(r.file)

			for len(batchChs) >= i.opt.EmbedBatchSize {
				if err := flush(batchChs[:i.opt.EmbedBatchSize]); err != nil {
					errCh <- err
					return
				}
				batchChs = batchChs[i.opt.EmbedBatchSize:]
			}
		}

		// Parsing finished; switch to embed stage start at 60%
		send(models.IndexProgress{
			Stage:          models.IndexStageEmbed,
			TotalFiles:     totalFiles,
			ParsedFiles:    parsedFiles,
			TotalChunks:    totalChunks,
			EmbeddedChunks: embeddedChunks,
			Percent:        0.6,
		})

		if err := flush(batchChs); err != nil {
			errCh <- err
			return
		}

		// Symbols upsert
		send(models.IndexProgress{
			Stage:          models.IndexStageSymbols,
			Percent:        0.95,
			Message:        "upserting symbols",
			TotalFiles:     totalFiles,
			ParsedFiles:    parsedFiles,
			TotalChunks:    totalChunks,
			EmbeddedChunks: embeddedChunks,
		})
		if err := i.sym.UpsertSymbols(allSyms); err != nil {
			errCh <- err
			return
		}

		// Done
		send(models.IndexProgress{
			Stage:          models.IndexStageDone,
			TotalFiles:     totalFiles,
			ParsedFiles:    parsedFiles,
			TotalChunks:    totalChunks,
			EmbeddedChunks: embeddedChunks,
			Percent:        1.0,
			Message:        "index completed",
		})
	}()

	return progCh, errCh
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
