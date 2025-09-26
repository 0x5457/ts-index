# LSP Integration Usage Examples

This document provides examples of how to use the LSP (Language Server Protocol) integration with vtsls for TypeScript code analysis.

## Prerequisites

1. Install vtsls language server:
```bash
npm install -g @vtsls/language-server
```

2. Check if vtsls is available:
```bash
./ts-index lsp health
```

## Command Line Usage

### 1. Analyze Symbol at Position

Analyze a symbol at a specific position in a TypeScript file:

```bash
# Basic symbol analysis with hover and definition info
./ts-index lsp analyze src/index.ts --project /path/to/project --line 10 --character 5 --hover --defs

# Include references as well
./ts-index lsp analyze src/index.ts --project /path/to/project --line 10 --character 5 --hover --defs --refs
```

### 2. Get Completion Items

Get code completion suggestions at a specific position:

```bash
./ts-index lsp completion src/index.ts --project /path/to/project --line 15 --character 10 --max-results 20
```

### 3. Search Workspace Symbols

Search for symbols across the entire workspace:

```bash
./ts-index lsp symbols --project /path/to/project --query "MyClass" --max-results 50
```

## HTTP API Usage

### 1. Start LSP Server

Start the HTTP server that exposes LSP functionality:

```bash
./ts-index lsp server --port 8080
```

### 2. API Endpoints

#### Analyze Symbol
```bash
curl -X POST http://localhost:8080/lsp/analyze-symbol \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_root": "/path/to/project",
    "file_path": "src/index.ts",
    "line": 10,
    "character": 5,
    "include_hover": true,
    "include_refs": false,
    "include_defs": true
  }'
```

#### Get Completion
```bash
curl -X POST http://localhost:8080/lsp/completion \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_root": "/path/to/project",
    "file_path": "src/index.ts",
    "line": 15,
    "character": 10,
    "max_results": 20
  }'
```

#### Search Symbols
```bash
curl -X POST http://localhost:8080/lsp/search-symbols \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_root": "/path/to/project",
    "query": "MyClass",
    "max_results": 50
  }'
```

#### Get Document Symbols
```bash
curl -X POST http://localhost:8080/lsp/document-symbols \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_root": "/path/to/project",
    "file_path": "src/index.ts"
  }'
```

#### Health Check
```bash
curl http://localhost:8080/lsp/health
```

## LLM Tool Integration

The LSP integration is designed to provide powerful tools for LLMs to analyze TypeScript code. Here are some example use cases:

### 1. Code Understanding
- Get hover information to understand what a symbol represents
- Find definitions to understand where symbols are declared
- Find references to understand how symbols are used

### 2. Code Navigation
- Navigate to definitions of functions, classes, interfaces
- Find all usages of a symbol across the codebase
- Search for symbols by name across the workspace

### 3. Code Completion
- Get intelligent completion suggestions based on TypeScript analysis
- Understand available methods and properties on objects
- Get parameter hints for function calls

### 4. Code Quality
- Access diagnostic information (errors, warnings)
- Understand type information through hover
- Validate code through LSP analysis

## Example TypeScript Project Structure

```
my-project/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts
│   ├── utils/
│   │   └── helpers.ts
│   └── types/
│       └── interfaces.ts
└── node_modules/
```

## Response Formats

### Symbol Analysis Response
```json
{
  "hover": {
    "contents": "function myFunction(param: string): number",
    "range": {
      "start": {"line": 10, "character": 0},
      "end": {"line": 10, "character": 10}
    }
  },
  "definitions": [
    {
      "uri": "file:///path/to/project/src/index.ts",
      "range": {
        "start": {"line": 5, "character": 9},
        "end": {"line": 5, "character": 19}
      }
    }
  ],
  "references": [],
  "error": ""
}
```

### Completion Response
```json
{
  "items": [
    {
      "label": "myMethod",
      "kind": 2,
      "detail": "(method) MyClass.myMethod(): void",
      "insert_text": "myMethod()"
    }
  ],
  "error": ""
}
```

### Symbol Search Response
```json
{
  "symbols": [
    {
      "name": "MyClass",
      "kind": 5,
      "location": {
        "uri": "file:///path/to/project/src/index.ts",
        "range": {
          "start": {"line": 10, "character": 0},
          "end": {"line": 20, "character": 1}
        }
      },
      "container_name": ""
    }
  ],
  "error": ""
}
```

## Integration with Existing Features

The LSP integration complements the existing tree-sitter based parsing and semantic search:

1. **Tree-sitter**: Fast structural parsing for code indexing
2. **LSP**: Rich semantic analysis with type information
3. **Semantic Search**: Vector-based search for code similarity
4. **Symbol Search**: Exact symbol matching with LSP enhancement

This multi-layered approach provides comprehensive code analysis capabilities for LLM applications.