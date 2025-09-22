package tsparser

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0x5457/ts-index/internal/models"
	"github.com/0x5457/ts-index/internal/parser"
	"github.com/0x5457/ts-index/internal/util"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tstypes "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type TSParser struct{}

func New() *TSParser { return &TSParser{} }

func (p *TSParser) ParseProject(root string) ([]models.Symbol, []models.CodeChunk, error) {
	var symbols []models.Symbol
	var chunks []models.CodeChunk
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "dist" ||
				d.Name() == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		if (!strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx")) ||
			strings.HasSuffix(path, ".d.ts") {
			return nil
		}
		syms, chs, perr := p.ParseFile(path)
		if perr != nil {
			return perr
		}
		symbols = append(symbols, syms...)
		chunks = append(chunks, chs...)
		return nil
	})
	if walkErr != nil {
		return nil, nil, walkErr
	}
	return symbols, chunks, nil
}

func (p *TSParser) ParseFile(path string) ([]models.Symbol, []models.CodeChunk, error) {
	code, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	parser := tree_sitter.NewParser()
	defer parser.Close()

	lang := tree_sitter.NewLanguage(tstypes.LanguageTypescript())
	languageName := "ts"
	if strings.HasSuffix(path, ".tsx") {
		lang = tree_sitter.NewLanguage(tstypes.LanguageTSX())
		languageName = "tsx"
	}
	if err := parser.SetLanguage(lang); err != nil {
		return nil, nil, err
	}

	tree := parser.Parse(code, nil)
	defer tree.Close()
	root := tree.RootNode()

	var symbols []models.Symbol
	var chunks []models.CodeChunk

	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		nt := n.Kind()
		switch nt {
		case "function_declaration":
			name := childIdentifier(n, code)
			appendDecl(
				&symbols,
				&chunks,
				path,
				languageName,
				nt,
				code,
				n,
				models.SymbolFunction,
				name,
			)
		case "class_declaration":
			name := childIdentifier(n, code)
			appendDecl(&symbols, &chunks, path, languageName, nt, code, n, models.SymbolClass, name)
		case "method_definition", "method_signature":
			name := childIdentifier(n, code)
			appendDecl(
				&symbols,
				&chunks,
				path,
				languageName,
				nt,
				code,
				n,
				models.SymbolMethod,
				name,
			)
		case "interface_declaration":
			name := childIdentifier(n, code)
			appendDecl(
				&symbols,
				&chunks,
				path,
				languageName,
				nt,
				code,
				n,
				models.SymbolInterface,
				name,
			)
		case "type_alias_declaration":
			name := childIdentifier(n, code)
			appendDecl(&symbols, &chunks, path, languageName, nt, code, n, models.SymbolType, name)
		case "enum_declaration":
			name := childIdentifier(n, code)
			appendDecl(&symbols, &chunks, path, languageName, nt, code, n, models.SymbolEnum, name)
		case "lexical_declaration",
			"variable_statement",
			"variable_declaration",
			"variable_declarator":
			collectVariables(n, path, languageName, code, &symbols, &chunks)
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)

	return symbols, chunks, nil
}

func childIdentifier(n *tree_sitter.Node, code []byte) string {
	// Prefer named field `name` if available
	if c := n.ChildByFieldName("name"); c != nil {
		return string(code[c.StartByte():c.EndByte()])
	}
	// Fallback to common identifier node kinds
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		kind := c.Kind()
		if kind == "identifier" || kind == "property_identifier" || kind == "type_identifier" {
			return string(code[c.StartByte():c.EndByte()])
		}
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		kind := c.Kind()
		if kind == "identifier" || kind == "property_identifier" || kind == "type_identifier" {
			return string(code[c.StartByte():c.EndByte()])
		}
	}
	return ""
}

func collectVariables(
	n *tree_sitter.Node,
	path, language string,
	code []byte,
	symbols *[]models.Symbol,
	chunks *[]models.CodeChunk,
) {
	if n.Kind() == "variable_declarator" {
		name := childIdentifier(n, code)
		appendDecl(symbols, chunks, path, language, n.Kind(), code, n, models.SymbolVariable, name)
		return
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		collectVariables(n.Child(i), path, language, code, symbols, chunks)
	}
}

func appendDecl(
	symbols *[]models.Symbol,
	chunks *[]models.CodeChunk,
	path, language, nodeType string,
	code []byte,
	n *tree_sitter.Node,
	kind models.SymbolKind,
	name string,
) {
	startLine := int32(n.StartPosition().Row) + 1
	endLine := int32(n.EndPosition().Row) + 1
	startByte := int32(n.StartByte())
	endByte := int32(n.EndByte())
	content := string(code[n.StartByte():n.EndByte()])
	sig := firstLine(content)
	doc := ""
	id := util.GenerateID(path, int(startLine), int(endLine), string(kind), name)
	*symbols = append(
		*symbols,
		models.Symbol{
			ID:        id,
			Name:      name,
			Kind:      kind,
			File:      path,
			Language:  language,
			NodeType:  nodeType,
			StartLine: startLine,
			EndLine:   endLine,
			StartByte: startByte,
			EndByte:   endByte,
			Docstring: doc,
		},
	)
	*chunks = append(
		*chunks,
		models.CodeChunk{
			ID:        id,
			File:      path,
			Language:  language,
			NodeType:  nodeType,
			StartLine: startLine,
			EndLine:   endLine,
			StartByte: startByte,
			EndByte:   endByte,
			Content:   content,
			Docstring: doc,
			Signature: sig,
			Kind:      kind,
			Name:      name,
		},
	)
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

var _ parser.Parser = (*TSParser)(nil)
