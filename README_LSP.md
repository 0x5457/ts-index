# LSP Integration for TypeScript Code Indexing Tool

This document describes the Language Server Protocol (LSP) integration implemented for your TypeScript code indexing tool, inspired by Zed's LanguageServer trait design.

## 🎯 Overview

The LSP integration provides rich semantic analysis capabilities for TypeScript/JavaScript code, complementing the existing tree-sitter based parsing and vector search features. This creates a comprehensive code analysis platform suitable for LLM applications.

## 🏗️ Architecture

### Core Components

1. **Language Server Interface** (`internal/lsp/language_server.go`)
   - Generic trait-based design inspired by Zed
   - Supports hover, completion, definitions, references, and symbol search
   - Extensible for multiple language servers

2. **LSP Client** (`internal/lsp/client.go`)
   - JSON-RPC 2.0 protocol implementation
   - Process management and communication
   - Async request/response handling

3. **TypeScript Support** (`internal/lsp/typescript.go`)
   - Factory pattern for TypeScript language servers
   - Support for both `vtsls` and `typescript-language-server`
   - Automatic fallback between servers

4. **Tools Layer** (`internal/lsp/tools.go`)
   - High-level API for LLM integration
   - Structured request/response types
   - Error handling and validation

5. **Service Layer** (`internal/lsp/service.go`)
   - HTTP endpoints for external access
   - RESTful API design
   - JSON request/response format

## 🚀 Features

### Language Server Support
- **vtsls**: Advanced TypeScript language server with enhanced features
- **typescript-language-server**: Standard TypeScript/JavaScript language server
- Automatic detection and fallback

### LSP Capabilities
- **Hover Information**: Type information, documentation, signatures
- **Go to Definition**: Navigate to symbol definitions
- **Find References**: Locate all symbol usages
- **Code Completion**: Intelligent code suggestions
- **Workspace Symbols**: Search symbols across the project
- **Document Symbols**: Get all symbols in a file

### Integration Methods
1. **Command Line Interface**: Direct CLI access to LSP features
2. **HTTP API**: RESTful endpoints for LLM integration
3. **Go API**: Direct Go package usage

## 📝 Usage Examples

### Installation

First, install a TypeScript language server:

```bash
# Option 1: vtsls (recommended)
npm install -g @vtsls/language-server

# Option 2: typescript-language-server
npm install -g typescript-language-server typescript
```

### Check Health

```bash
./ts-index lsp health
```

### Command Line Usage

**Analyze Symbol at Position:**
```bash
./ts-index lsp analyze src/index.ts \
  --project /path/to/typescript/project \
  --line 10 --character 5 \
  --hover --defs --refs
```

**Get Code Completions:**
```bash
./ts-index lsp completion src/index.ts \
  --project /path/to/typescript/project \
  --line 15 --character 10 \
  --max-results 20
```

**Search Workspace Symbols:**
```bash
./ts-index lsp symbols \
  --project /path/to/typescript/project \
  --query "MyClass" \
  --max-results 50
```

### HTTP API Usage

**Start the Server:**
```bash
./ts-index lsp server --port 8080
```

**API Endpoints:**

1. **Analyze Symbol**
```bash
curl -X POST http://localhost:8080/lsp/analyze-symbol \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_root": "/path/to/project",
    "file_path": "src/index.ts",
    "line": 10,
    "character": 5,
    "include_hover": true,
    "include_refs": true,
    "include_defs": true
  }'
```

2. **Get Completions**
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

3. **Search Symbols**
```bash
curl -X POST http://localhost:8080/lsp/search-symbols \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_root": "/path/to/project",
    "query": "MyClass",
    "max_results": 50
  }'
```

4. **Health Check**
```bash
curl http://localhost:8080/lsp/health
```

## 🔧 Configuration

### Language Server Configuration

The tool automatically configures language servers with optimal settings:

- **Auto-imports**: Enabled for better completion experience
- **Inlay hints**: Configured for parameter names and types
- **Fuzzy matching**: Enhanced completion matching (vtsls)
- **Content format**: Supports both markdown and plaintext

### Environment Variables

Set these environment variables if needed:
- `NODE_PATH`: For custom Node.js module resolution
- `TYPESCRIPT_LIB`: Custom TypeScript library path

## 🏆 Benefits for LLM Applications

### 1. Rich Context Understanding
- Get precise type information for any symbol
- Access comprehensive documentation from hover
- Understand symbol relationships through references

### 2. Intelligent Code Navigation
- Navigate codebases programmatically
- Find symbol definitions across files
- Locate all usages of specific symbols

### 3. Enhanced Code Completion
- Get contextually relevant suggestions
- Access auto-import capabilities
- Understand available methods and properties

### 4. Multi-layered Analysis
- **Tree-sitter**: Fast structural parsing
- **LSP**: Rich semantic analysis
- **Vector search**: Semantic similarity
- **Combined**: Comprehensive understanding

## 🔍 Example Response Formats

### Symbol Analysis Response
```json
{
  "hover": {
    "contents": "class Calculator\nA sample TypeScript class for testing LSP functionality",
    "range": {
      "start": {"line": 3, "character": 13},
      "end": {"line": 3, "character": 23}
    }
  },
  "definitions": [
    {
      "uri": "file:///path/to/src/index.ts",
      "range": {
        "start": {"line": 3, "character": 13},
        "end": {"line": 3, "character": 23}
      }
    }
  ],
  "references": [
    {
      "uri": "file:///path/to/src/index.ts",
      "range": {
        "start": {"line": 44, "character": 13},
        "end": {"line": 44, "character": 23}
      }
    }
  ]
}
```

### Completion Response
```json
{
  "items": [
    {
      "label": "add",
      "kind": 2,
      "detail": "(method) Calculator.add(a: number, b: number): number",
      "insert_text": "add(${1:a}, ${2:b})"
    },
    {
      "label": "multiply", 
      "kind": 2,
      "detail": "(method) Calculator.multiply(a: number, b: number): number",
      "insert_text": "multiply(${1:a}, ${2:b})"
    }
  ]
}
```

## 🛠️ Implementation Details

### Design Patterns Used
- **Factory Pattern**: Language server creation
- **Manager Pattern**: Multi-workspace handling
- **Trait/Interface Pattern**: Generic language server interface
- **Builder Pattern**: Configuration construction

### Error Handling
- Graceful degradation when language servers are unavailable
- Timeout handling for LSP requests
- Process lifecycle management
- Comprehensive error reporting

### Performance Considerations
- Lazy server initialization
- Connection pooling for multiple workspaces
- Efficient JSON-RPC message handling
- Background process management

## 🔮 Future Enhancements

1. **Additional Language Servers**
   - Python (Pylsp, Pyright)
   - Go (gopls)
   - Rust (rust-analyzer)

2. **Advanced Features**
   - Code actions and quick fixes
   - Rename refactoring
   - Formatting and linting integration

3. **Performance Optimizations**
   - Incremental synchronization
   - Caching strategies
   - Connection pooling

4. **Enhanced LLM Integration**
   - Semantic code search
   - Context-aware code generation
   - Intelligent code review

## 📚 API Reference

### Go Package Usage

```go
import "github.com/0x5457/ts-index/internal/lsp"

// Create TypeScript features
tsFeatures := lsp.NewTypeScriptFeatures()
defer tsFeatures.Cleanup()

// Analyze symbol
hover, err := tsFeatures.GetHoverInfo(
    ctx, 
    workspaceRoot, 
    filePath, 
    line, 
    character,
)

// Get completions
completions, err := tsFeatures.GetCompletion(
    ctx,
    workspaceRoot,
    filePath,
    line,
    character,
)
```

### HTTP API Reference

See the example usage above for complete HTTP API documentation.

## 🤝 Contributing

The LSP integration is designed to be extensible. To add support for new language servers:

1. Implement the `LanguageServer` interface
2. Create a factory for your language server
3. Add configuration and initialization logic
4. Update the health check and documentation

This implementation provides a solid foundation for rich code analysis capabilities in your TypeScript indexing tool, making it an ideal platform for LLM-powered code understanding and generation tasks.