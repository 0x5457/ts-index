package models

import "github.com/0x5457/ts-index/internal/lsp"

// Use SymbolKind from lsp package
type SymbolKind = lsp.SymbolKind

// Symbol kind constants for backward compatibility
const (
	SymbolFunction  = lsp.SymbolKindFunction
	SymbolMethod    = lsp.SymbolKindMethod
	SymbolClass     = lsp.SymbolKindClass
	SymbolInterface = lsp.SymbolKindInterface
	SymbolType      = lsp.SymbolKindStruct // Using struct for type
	SymbolEnum      = lsp.SymbolKindEnum
	SymbolVariable  = lsp.SymbolKindVariable
)

// StringToSymbolKind converts string to SymbolKind
func StringToSymbolKind(s string) SymbolKind {
	switch s {
	case "function":
		return SymbolFunction
	case "method":
		return SymbolMethod
	case "class":
		return SymbolClass
	case "interface":
		return SymbolInterface
	case "type":
		return SymbolType
	case "enum":
		return SymbolEnum
	case "variable":
		return SymbolVariable
	default:
		return lsp.SymbolKindVariable // default fallback
	}
}

type Symbol struct {
	ID        string
	Name      string
	Kind      SymbolKind
	File      string
	Language  string
	NodeType  string
	StartLine int32
	EndLine   int32
	StartByte int32
	EndByte   int32
	Docstring string
}

type CodeChunk struct {
	ID        string
	File      string
	Language  string
	NodeType  string
	StartLine int32
	EndLine   int32
	StartByte int32
	EndByte   int32
	Content   string
	Docstring string
	Signature string
	Kind      SymbolKind
	Name      string
}

type SemanticHit struct {
	Chunk CodeChunk
	Score float32
}

type SymbolHit struct {
	Symbol Symbol
}

// Index progress and stages
type IndexStage string

const (
	IndexStageScan    IndexStage = "scan"
	IndexStageParse   IndexStage = "parse"
	IndexStageEmbed   IndexStage = "embed"
	IndexStageSymbols IndexStage = "symbols"
	IndexStageDone    IndexStage = "done"
)

// IndexProgress represents streaming progress updates for indexing
type IndexProgress struct {
	Stage          IndexStage
	TotalFiles     int
	ParsedFiles    int
	TotalChunks    int
	EmbeddedChunks int
	CurrentFile    string
	Message        string
	Percent        float32
}

// LSPHoverInfo represents hover information from LSP
type LSPHoverInfo struct {
	Contents string     `json:"contents"`
	Range    *lsp.Range `json:"range,omitempty"`
}

// Use Location from lsp package
type Location = lsp.Location

// LSPCompletionItem represents a completion item from LSP
type LSPCompletionItem struct {
	Label      string `json:"label"`
	Kind       *int   `json:"kind,omitempty"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
}

// LSPSymbolInfo represents symbol information from LSP
type LSPSymbolInfo struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

// LSPDiagnostic represents a diagnostic from LSP
type LSPDiagnostic struct {
	Range    lsp.Range `json:"range"`
	Severity *int      `json:"severity,omitempty"`
	Code     string    `json:"code,omitempty"`
	Source   string    `json:"source,omitempty"`
	Message  string    `json:"message"`
}

// EnhancedSymbol extends Symbol with LSP information
type EnhancedSymbol struct {
	Symbol
	LSPHover       *LSPHoverInfo   `json:"lsp_hover,omitempty"`
	LSPDefinitions []Location      `json:"lsp_definitions,omitempty"`
	LSPReferences  []Location      `json:"lsp_references,omitempty"`
	LSPDiagnostics []LSPDiagnostic `json:"lsp_diagnostics,omitempty"`
}
