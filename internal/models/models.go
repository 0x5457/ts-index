package models

type SymbolKind string

const (
	SymbolFunction  SymbolKind = "function"
	SymbolMethod    SymbolKind = "method"
	SymbolClass     SymbolKind = "class"
	SymbolInterface SymbolKind = "interface"
	SymbolType      SymbolKind = "type"
	SymbolEnum      SymbolKind = "enum"
	SymbolVariable  SymbolKind = "variable"
)

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
	Contents string `json:"contents"`
	Range    *Range `json:"range,omitempty"`
}

// LSPLocation represents a location from LSP
type LSPLocation struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Range represents a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// LSPCompletionItem represents a completion item from LSP
type LSPCompletionItem struct {
	Label      string `json:"label"`
	Kind       *int   `json:"kind,omitempty"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
}

// LSPSymbolInfo represents symbol information from LSP
type LSPSymbolInfo struct {
	Name          string      `json:"name"`
	Kind          int         `json:"kind"`
	Location      LSPLocation `json:"location"`
	ContainerName string      `json:"containerName,omitempty"`
}

// LSPDiagnostic represents a diagnostic from LSP
type LSPDiagnostic struct {
	Range    Range  `json:"range"`
	Severity *int   `json:"severity,omitempty"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// EnhancedSymbol extends Symbol with LSP information
type EnhancedSymbol struct {
	Symbol
	LSPHover       *LSPHoverInfo     `json:"lsp_hover,omitempty"`
	LSPDefinitions []LSPLocation     `json:"lsp_definitions,omitempty"`
	LSPReferences  []LSPLocation     `json:"lsp_references,omitempty"`
	LSPDiagnostics []LSPDiagnostic   `json:"lsp_diagnostics,omitempty"`
}
