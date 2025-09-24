package tsparser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0x5457/ts-index/internal/models"
	p "github.com/0x5457/ts-index/internal/parser/tsparser"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func Test_TSParser_ParseProject_TS_and_TSX(t *testing.T) {
	tmp := t.TempDir()
	ts := `
// interface
interface I { x: number }
// type alias
type Alias = string
// enum
export enum E { A, B }
// class with method
export class C {
  m(): void { }
}
// function
export function f(x: number): number { return x }
// variable
const v = 1
`
	tsx := `
export function Component(): JSX.Element { return <div/> }
`
	writeFile(t, tmp, "a.ts", ts)
	writeFile(t, tmp, "b.tsx", tsx)

	parser := p.New()
	symbols, chunks, err := parser.ParseProject(tmp)
	if err != nil {
		t.Fatalf("ParseProject error: %v", err)
	}
	if len(symbols) == 0 || len(chunks) == 0 {
		t.Fatalf("expected non-empty symbols and chunks")
	}
	// basic sanity checks
	kindCount := map[models.SymbolKind]int{}
	langs := map[string]bool{}
	for i, s := range symbols {
		if s.Name == "" {
			t.Fatalf("symbol name empty at %d", i)
		}
		if s.StartLine <= 0 || s.EndLine < s.StartLine {
			t.Fatalf("invalid lines for %s", s.Name)
		}
		if s.NodeType == "" {
			t.Fatalf("empty node type for %s", s.Name)
		}
		if s.Language == "" {
			t.Fatalf("empty language for %s", s.Name)
		}
		kindCount[s.Kind]++
		langs[s.Language] = true
	}
	if !langs["ts"] || !langs["tsx"] {
		t.Fatalf("expected both ts and tsx languages, got %v", langs)
	}
	if kindCount[models.SymbolFunction] == 0 {
		t.Fatalf("expected at least one function")
	}
	if kindCount[models.SymbolClass] == 0 {
		t.Fatalf("expected at least one class")
	}
	if kindCount[models.SymbolMethod] == 0 {
		t.Fatalf("expected at least one method")
	}
	if kindCount[models.SymbolInterface] == 0 {
		t.Fatalf("expected at least one interface")
	}
	if kindCount[models.SymbolType] == 0 {
		t.Fatalf("expected at least one type alias")
	}
	if kindCount[models.SymbolEnum] == 0 {
		t.Fatalf("expected at least one enum")
	}
	if kindCount[models.SymbolVariable] == 0 {
		t.Fatalf("expected at least one variable")
	}
}

func Test_TSParser_Docstrings(t *testing.T) {
	tmp := t.TempDir()
	code := `
/**
 * add jsdoc
 * multi line
 */
export function add(a: number, b: number): number { return a + b } // add trailing

// line doc 1
// line doc 2
export const x = 1

export function hello(): void {} // hello doc

export const y = 2 /* y block doc */

export function paramDoc(a: number /* a is value */): number { return a } // end doc
`
	writeFile(t, tmp, "doc.ts", code)

	parser := p.New()
	symbols, chunks, err := parser.ParseFile(filepath.Join(tmp, "doc.ts"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	expect := map[string]string{
		"add":      "add jsdoc\nmulti line\nadd trailing",
		"x":        "line doc 1\nline doc 2",
		"hello":    "hello doc",
		"y":        "y block doc",
		"paramDoc": "a is value\nend doc",
	}

	// Verify symbols
	for name, want := range expect {
		found := false
		for _, s := range symbols {
			if s.Name == name {
				found = true
				if s.Docstring != want {
					t.Fatalf(
						"symbol %s docstring mismatch:\nwant=\n%q\nget =\n%q",
						name,
						want,
						s.Docstring,
					)
				}
				break
			}
		}
		if !found {
			t.Fatalf("symbol %s not found", name)
		}
	}

	// Verify chunks
	for name, want := range expect {
		found := false
		for _, c := range chunks {
			if c.Name == name {
				found = true
				if c.Docstring != want {
					t.Fatalf(
						"chunk %s docstring mismatch:\nwant=\n%q\nget =\n%q",
						name,
						want,
						c.Docstring,
					)
				}
				break
			}
		}
		if !found {
			t.Fatalf("chunk %s not found", name)
		}
	}
}
