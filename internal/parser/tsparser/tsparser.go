package tsparser

import (
	"bytes"
	"fmt"
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
	// Only add symbol/chunk for the exact declarator node to avoid duplicates.
	if n.Kind() != "variable_declarator" {
		return
	}
	name := childIdentifier(n, code)
	appendDecl(symbols, chunks, path, language, n.Kind(), code, n, models.SymbolVariable, name)
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
	doc := extractDocstring(code, n)
	id := util.GenerateID(path, int(startLine), int(endLine), fmt.Sprint(rune(kind)), name)
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

// extractDocstring tries to capture the leading doc comment for a node.
// It supports JSDoc-style block comments (/** ... */) and consecutive
// single-line comments (// ...), immediately preceding the node with only
// whitespace in between.
func extractDocstring(code []byte, n *tree_sitter.Node) string {
	if n == nil {
		return ""
	}
	start := int(n.StartByte())
	if start < 0 {
		return ""
	}

	var parts []string

	// Leading: consider only the bytes before the declaration line start
	if start > 0 {
		lineStart := bytes.LastIndexByte(code[:start], '\n') + 1
		pre := code[:lineStart]
		pre = trimRightWhitespace(pre)
		if len(pre) > 0 {
			// Find the nearest preceding block comment that begins at line start and is JSDoc
			closeIdx := bytes.LastIndex(pre, []byte("*/"))
			for closeIdx >= 0 {
				openIdx := bytes.LastIndex(pre[:closeIdx], []byte("/*"))
				if openIdx < 0 {
					break
				}
				openLineStart := bytes.LastIndexByte(pre[:openIdx], '\n') + 1
				beginsAtLine := len(bytes.TrimSpace(pre[openLineStart:openIdx])) == 0
				raw := pre[openIdx : closeIdx+2]
				isJSDoc := bytes.HasPrefix(bytes.TrimLeft(raw, " \t\r\n"), []byte("/**"))
				tail := bytes.TrimSpace(pre[closeIdx+2:])
				if beginsAtLine && isJSDoc && (len(tail) == 0 || isOnlyTSModifiers(tail)) {
					if s := cleanBlockComment(raw); s != "" {
						parts = append(parts, s)
					}
					break
				}
				// move to previous block (before this one's open)
				closeIdx = bytes.LastIndex(pre[:openIdx], []byte("*/"))
			}

			// Try consecutive //-style lines immediately preceding
			lines := collectLineCommentsBeforeLine(code, lineStart)
			if len(lines) > 0 {
				for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
					lines[i], lines[j] = lines[j], lines[i]
				}
				s := strings.TrimSpace(strings.Join(lines, "\n"))
				if s != "" {
					parts = append(parts, s)
				}
			}
		}
	}

	// Inline on start line
	if s := extractInlineOnStartLine(code, n); s != "" {
		parts = append(parts, s)
	}

	// Trailing on end line
	if s := extractTrailingOnEndLine(code, n); s != "" {
		// If we already have block/line doc before, append trailing only if it is not duplicate
		if len(parts) == 0 || parts[len(parts)-1] != s {
			parts = append(parts, s)
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func trimRightWhitespace(b []byte) []byte {
	i := len(b) - 1
	for i >= 0 {
		c := b[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		i--
	}
	return b[:i+1]
}

func cleanBlockComment(raw []byte) string {
	s := string(raw)
	// Remove /*, /** and */
	s = strings.TrimPrefix(s, "/**")
	s = strings.TrimPrefix(s, "/*")
	if idx := strings.LastIndex(s, "*/"); idx >= 0 {
		s = s[:idx]
	}
	// Split lines and remove leading * and single leading space after *
	out := make([]string, 0, 8)
	for _, line := range strings.Split(s, "\n") {
		l := strings.TrimRight(line, " \t\r")
		l = strings.TrimLeft(l, " \t")
		if strings.HasPrefix(l, "*") {
			l = strings.TrimPrefix(l, "*")
			if len(l) > 0 && (l[0] == ' ' || l[0] == '\t') {
				l = l[1:]
			}
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func collectLineCommentsBeforeLine(code []byte, lineStart int) []string {
	var lines []string
	if lineStart <= 0 || lineStart > len(code) {
		return lines
	}
	// j points to the newline ending the previous line
	j := lineStart - 1
	for j >= 0 {
		prevLineStart := bytes.LastIndexByte(code[:j], '\n') + 1
		line := code[prevLineStart : j+1]
		// Stop if blank line
		if len(bytes.TrimSpace(line)) == 0 {
			break
		}
		ltrim := bytes.TrimLeft(line, " \t")
		if !bytes.HasPrefix(ltrim, []byte("//")) {
			break
		}
		content := string(bytes.TrimSpace(bytes.TrimPrefix(ltrim, []byte("//"))))
		lines = append(lines, content)
		if prevLineStart == 0 {
			break
		}
		j = prevLineStart - 1
	}
	return lines
}

// isOnlyTSModifiers reports whether b contains only TypeScript declaration modifiers
// and whitespace, e.g., export, default, async, declare, abstract, readonly.
func isOnlyTSModifiers(b []byte) bool {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return true
	}
	// Split by spaces; if any token contains non-letters (like identifier), return false
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return true
	}
	allowed := map[string]struct{}{
		"export":    {},
		"default":   {},
		"async":     {},
		"declare":   {},
		"abstract":  {},
		"readonly":  {},
		"public":    {},
		"private":   {},
		"protected": {},
		"static":    {},
	}
	for _, t := range tokens {
		if _, ok := allowed[t]; !ok {
			return false
		}
	}
	return true
}

func extractInlineOnStartLine(code []byte, n *tree_sitter.Node) string {
	start := int(n.StartByte())
	end := int(n.EndByte())
	if start < 0 || start >= len(code) {
		return ""
	}
	if end < 0 || end > len(code) || end < start {
		end = len(code)
	}
	// Only search within the node span to capture true inline block comments (e.g., parameter comments)
	// We intentionally ignore '//' here to avoid swallowing trailing line comments which may also fall within
	// the node span depending on the grammar.
	seg := code[start:end]
	// up to newline
	nl := bytes.IndexByte(seg, '\n')
	if nl >= 0 {
		seg = seg[:nl]
	}
	idxBlock := bytes.Index(seg, []byte("/*"))
	if idxBlock < 0 {
		return ""
	}
	rest := seg[idxBlock+2:]
	close := bytes.Index(rest, []byte("*/"))
	if close < 0 {
		return ""
	}
	raw := append([]byte("/*"), rest[:close]...)
	raw = append(raw, []byte("*/")...)
	return strings.TrimSpace(cleanBlockComment(raw))
}

func extractTrailingOnEndLine(code []byte, n *tree_sitter.Node) string {
	end := int(n.EndByte())
	if end < 0 || end > len(code) {
		return ""
	}
	// Reference position for locating the end line: use end-1 to stay on the line where the node ends
	ref := end
	if ref > 0 {
		ref = ref - 1
	}
	// Start of the line where the node ends
	lineStart := bytes.LastIndexByte(code[:ref], '\n') + 1
	// End of that line
	idx := bytes.IndexByte(code[ref:], '\n')
	var lineEnd int
	if idx >= 0 {
		lineEnd = ref + idx
	} else {
		lineEnd = len(code)
	}
	if lineEnd < lineStart {
		return ""
	}
	line := code[lineStart:lineEnd]
	// Position within the line where to start searching for trailing comment
	posInLine := end - lineStart
	if posInLine < 0 {
		posInLine = 0
	}
	if posInLine > len(line) {
		posInLine = len(line)
	}
	seg := line[posInLine:]
	if len(bytes.TrimSpace(seg)) == 0 {
		// still try whole line to catch cases where end points at newline
		seg = line
	}
	// Prefer the last // anywhere on the end line
	if idx := bytes.LastIndex(seg, []byte("//")); idx >= 0 {
		return strings.TrimSpace(string(seg[idx+2:]))
	}
	// Fallback: last /* ... */ on the end line
	if startIdx := bytes.LastIndex(seg, []byte("/*")); startIdx >= 0 {
		if endIdx := bytes.Index(seg[startIdx+2:], []byte("*/")); endIdx >= 0 {
			raw := append([]byte("/*"), seg[startIdx+2:startIdx+2+endIdx]...)
			raw = append(raw, []byte("*/")...)
			return strings.TrimSpace(cleanBlockComment(raw))
		}
	}
	return ""
}
