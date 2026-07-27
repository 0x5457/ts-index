# ts-index

Una potente herramienta de indexación y búsqueda de código TypeScript con soporte para Language Server Protocol e integración con MCP (Model Context Protocol).

## Características Principales

- **Búsqueda Semántica**: Búsqueda de código avanzada utilizando embeddings para consultas en lenguaje natural
- **Búsqueda de Símbolos**: Coincidencia y búsqueda exacta de nombres de símbolos
- **Integración LSP**: Soporte de Language Server Protocol para el análisis de TypeScript
- **Servidor MCP**: Expone las capacidades de indexación y búsqueda a través del Model Context Protocol
- **Base de Datos Vectorial**: Almacenamiento basado en SQLite con capacidades de búsqueda vectorial
- **Múltiples Transportes**: Soporte para protocolos de comunicación stdio, HTTP y SSE

## Stack Tecnológico

- **TypeScript Parser**: [tree-sitter-typescript](https://github.com/tree-sitter/tree-sitter-typescript) - Gramáticas de TypeScript y TSX para tree-sitter
- **Language Server**: [yioneko/vtsls](https://github.com/yioneko/vtsls) / [typescript-language-server](https://github.com/typescript-language-server/typescript-language-server) - Wrapper LSP para TypeScript
- **Base de Datos Vectorial**: [asg017/sqlite-vec](https://github.com/asg017/sqlite-vec) - Extensión de búsqueda vectorial para SQLite

## Uso

### Indexar un proyecto de TypeScript

```bash
ts-index index --project /path/to/project --db /path/to/index.db
```

### Buscar código semánticamente

```bash
ts-index search "function to parse JSON" --project /path/to/project --db /path/to/index.db
```

### Buscar por nombre exacto de símbolo

```bash
ts-index search "parseJSON" --symbol --db /path/to/index.db
```

### Comandos de Language Server Protocol

```bash
# Analizar símbolo en una posición
ts-index lsp analyze src/utils.ts --project /path/to/project --line 10 --character 5

# Obtener completados de código
ts-index lsp completion src/utils.ts --project /path/to/project --line 10 --character 5

# Buscar símbolos del espacio de trabajo
ts-index lsp symbols --project /path/to/project --query "parse"

# Instalar language server
ts-index lsp install vtsls

# Verificar estado del LSP
ts-index lsp health
```

### Ejecutar servidor MCP

```bash
# modo stdio (por defecto)
ts-index mcp --project /path/to/project --db /path/to/index.db

# modo HTTP
ts-index mcp --transport http --address :8080 --db /path/to/index.db

# modo SSE
ts-index mcp --transport sse --address :8080 --db /path/to/index.db
```

## Desarrollo

### Comandos

```bash
# Ejecutar linter
make lint

# Ejecutar linter con auto-corrección
make lint-fix

# Ejecutar pruebas
make test
```

### Construcción

```bash
go build -o bin/ts-index ./cmd/ts-index
```
