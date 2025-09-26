# Dependency Injection with Uber Fx

This project has been refactored to use [Uber's Fx](https://uber-go.github.io/fx/) dependency injection framework following best practices for clean, maintainable, and testable code.

## Architecture Overview

The application is now organized into modular Fx components located in `internal/fx/`:

### Core Modules

- **ConfigModule** (`config.go`): Provides application configuration with defaults
- **ParserModule** (`parser.go`): Provides TypeScript parser instances
- **EmbeddingsModule** (`embeddings.go`): Provides embedding services
- **StorageModule** (`storage.go`): Provides symbol and vector storage
- **SearchModule** (`search.go`): Provides semantic search services
- **IndexerModule** (`indexer.go`): Provides code indexing services
- **MCPModule** (`mcp.go`): Provides MCP server with lifecycle management
- **CommandModule** (`commands.go`): Provides command runner for CLI operations

### Fx Best Practices Implemented

1. **Modular Design**: Each major component group is organized into separate modules
2. **Parameter Structs**: Uses `fx.In` embedded structs for clean dependency injection
3. **Lifecycle Management**: Proper startup and shutdown hooks for resources
4. **Configuration Supply**: Uses `fx.Supply` and `fx.Annotate` for configuration injection
5. **Interface-Based Design**: Maintains interface-based dependency injection
6. **Testability**: Comprehensive test coverage for all modules

## Key Components

### Configuration Structure

```go
type Config struct {
    DBPath          string
    EmbedURL        string  
    VectorDimension int
    Project         string // Optional project path for pre-indexing
}
```

### Dependency Injection Pattern

Components use the `fx.In` pattern for parameter injection:

```go
type StorageParams struct {
    fx.In
    Config *Config
}

func NewSymbolStore(params StorageParams) (storage.SymbolStore, error) {
    return sqlite.New(params.Config.DBPath)
}
```

### Command Runner

The `CommandRunner` provides a clean interface for executing different application commands:

```go
type CommandRunner struct {
    config        *Config
    searchService *search.Service
    indexer       indexer.Indexer
    mcpServer     *server.MCPServer
}
```

## Usage Examples

### Creating an Fx App

```go
app := fx.New(
    appfx.AppModule,
    fx.Supply(
        fx.Annotate(dbPath, fx.ResultTags(`name:"dbPath"`)),
        fx.Annotate(embedURL, fx.ResultTags(`name:"embedURL"`)),
        fx.Annotate(project, fx.ResultTags(`name:"project"`)),
    ),
    fx.Invoke(func(runner *appfx.CommandRunner) error {
        return runner.RunIndex(cmd.Context(), projectPath)
    }),
)
```

### Lifecycle Management

The MCP server includes proper lifecycle hooks:

```go
func (m *MCPLifecycle) Start(ctx context.Context) error {
    if m.config.Project != "" {
        return m.indexer.IndexProject(m.config.Project)
    }
    return nil
}
```

## Command Integration

All CLI commands now use Fx for dependency injection:

- **Index Command**: Creates Fx app with indexing dependencies
- **MCP Command**: Uses Fx for server lifecycle management
- **Search Command**: Uses existing MCP client (already well-structured)

## Testing

Comprehensive tests are provided in `internal/fx/fx_test.go`:

- Module-level testing for each component
- Integration testing with full app assembly
- Proper resource cleanup and lifecycle management

Run tests with:
```bash
make test
```

## Benefits of This Implementation

1. **Clean Separation of Concerns**: Each module has a single responsibility
2. **Easy Testing**: Components can be easily mocked and tested in isolation
3. **Configuration Management**: Centralized config with proper defaults
4. **Resource Management**: Proper lifecycle hooks for cleanup
5. **Maintainability**: Clear dependency graph and modular structure
6. **Extensibility**: Easy to add new components and modify existing ones

## Development Workflow

1. **Lint and Format**: `make lint-fix`
2. **Run Tests**: `make test`
3. **Build**: `go build ./...`

Both `make lint-fix` and `make test` pass successfully, ensuring code quality and correctness.

## Migration Notes

The original factory pattern has been completely replaced with the new modular Fx architecture for better maintainability and cleaner dependency management.

## Future Enhancements

- Add configuration validation with Fx hooks
- Implement graceful shutdown for long-running operations  
- Add metrics and monitoring integration
- Expand test coverage for edge cases
- Add configuration hot-reloading capabilities