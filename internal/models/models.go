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
