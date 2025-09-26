package lsp

import (
	"encoding/json"
)

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a resource
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier represents a reference to a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentPositionParams represents a parameter literal used in requests to pass text document position
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Hover represents the result of a hover request
type Hover struct {
	Contents json.RawMessage `json:"contents"`
	Range    *Range          `json:"range,omitempty"`
}

// CompletionItem represents a completion item
type CompletionItem struct {
	Label         string           `json:"label"`
	Kind          *CompletionKind  `json:"kind,omitempty"`
	Detail        *string          `json:"detail,omitempty"`
	Documentation json.RawMessage  `json:"documentation,omitempty"`
	InsertText    *string          `json:"insertText,omitempty"`
	TextEdit      *TextEdit        `json:"textEdit,omitempty"`
}

// CompletionKind represents the kind of a completion item
type CompletionKind int

const (
	CompletionKindText          CompletionKind = 1
	CompletionKindMethod        CompletionKind = 2
	CompletionKindFunction      CompletionKind = 3
	CompletionKindConstructor   CompletionKind = 4
	CompletionKindField         CompletionKind = 5
	CompletionKindVariable      CompletionKind = 6
	CompletionKindClass         CompletionKind = 7
	CompletionKindInterface     CompletionKind = 8
	CompletionKindModule        CompletionKind = 9
	CompletionKindProperty      CompletionKind = 10
	CompletionKindUnit          CompletionKind = 11
	CompletionKindValue         CompletionKind = 12
	CompletionKindEnum          CompletionKind = 13
	CompletionKindKeyword       CompletionKind = 14
	CompletionKindSnippet       CompletionKind = 15
	CompletionKindColor         CompletionKind = 16
	CompletionKindFile          CompletionKind = 17
	CompletionKindReference     CompletionKind = 18
	CompletionKindFolder        CompletionKind = 19
	CompletionKindEnumMember    CompletionKind = 20
	CompletionKindConstant      CompletionKind = 21
	CompletionKindStruct        CompletionKind = 22
	CompletionKindEvent         CompletionKind = 23
	CompletionKindOperator      CompletionKind = 24
	CompletionKindTypeParameter CompletionKind = 25
)

// TextEdit represents a textual edit applicable to a text document
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// CompletionList represents a list of completion items
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// Diagnostic represents a diagnostic, such as a compiler error or warning
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity *DiagnosticSeverity `json:"severity,omitempty"`
	Code     json.RawMessage    `json:"code,omitempty"`
	Source   *string            `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// DiagnosticSeverity represents the severity of a diagnostic
type DiagnosticSeverity int

const (
	DiagnosticSeverityError       DiagnosticSeverity = 1
	DiagnosticSeverityWarning     DiagnosticSeverity = 2
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	DiagnosticSeverityHint        DiagnosticSeverity = 4
)

// SymbolInformation represents information about programming constructs like variables, classes, interfaces etc.
type SymbolInformation struct {
	Name          string       `json:"name"`
	Kind          SymbolKind   `json:"kind"`
	Deprecated    *bool        `json:"deprecated,omitempty"`
	Location      Location     `json:"location"`
	ContainerName *string      `json:"containerName,omitempty"`
}

// SymbolKind represents the kind of a symbol
type SymbolKind int

const (
	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26
)

// WorkspaceSymbolParams represents the parameters of a workspace symbol request
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}