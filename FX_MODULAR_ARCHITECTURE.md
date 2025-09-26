# Fx 模块化架构重构

本项目已经进行了系统性的重构，采用了真正的模块化设计，每个功能模块都有自己独立的 `fx` 包。这符合最佳实践，避免了将所有模块混合在一个 `fx` 包中的问题。

## 🏗️ 新的模块化结构

### 核心模块包

每个功能模块都有自己的 `fx` 包：

#### 1. **配置模块** (`internal/config/configfx/`)
```go
package configfx

type Config struct {
    DBPath          string
    EmbedURL        string
    VectorDimension int
    Project         string
}
```
- 提供应用程序配置管理
- 支持默认值设置
- 使用 `fx.Supply` 进行配置注入

#### 2. **解析器模块** (`internal/parser/parserfx/`)
```go
package parserfx

func NewParser() parser.Parser {
    return tsparser.New()
}
```
- 提供 TypeScript 解析器实例
- 无依赖的简单模块

#### 3. **嵌入模块** (`internal/embeddings/embeddingsfx/`)
```go
package embeddingsfx

type Params struct {
    fx.In
    Config *configfx.Config
}
```
- 依赖配置模块获取嵌入服务URL
- 提供向量嵌入服务

#### 4. **存储模块** (`internal/storage/storagefx/`)
```go
package storagefx

type Params struct {
    fx.In
    Config *configfx.Config
}
```
- 提供符号存储和向量存储
- 依赖配置模块获取数据库路径
- 包含错误处理

#### 5. **搜索模块** (`internal/search/searchfx/`)
```go
package searchfx

type Params struct {
    fx.In
    Embedder  embeddings.Embedder
    VecStore  storage.VectorStore
}
```
- 组合嵌入服务和向量存储
- 提供语义搜索功能

#### 6. **索引器模块** (`internal/indexer/indexerfx/`)
```go
package indexerfx

type Params struct {
    fx.In
    Parser   parser.Parser
    Embedder embeddings.Embedder
    SymStore storage.SymbolStore
    VecStore storage.VectorStore
}
```
- 组合所有核心依赖
- 提供代码索引管道

#### 7. **MCP 模块** (`internal/mcp/mcpfx/`)
```go
package mcpfx

type Lifecycle struct {
    server  *server.MCPServer
    indexer indexer.Indexer
    config  *configfx.Config
}
```
- 提供 MCP 服务器和生命周期管理
- 支持项目预索引
- 处理启动和关闭钩子

#### 8. **命令模块** (`cmd/cmdsfx/`)
```go
package cmdsfx

type CommandRunner struct {
    config        *configfx.Config
    searchService *search.Service
    indexer       indexer.Indexer
    mcpServer     *server.MCPServer
}
```
- 提供 CLI 命令执行逻辑
- 统一的命令接口

#### 9. **应用模块** (`internal/app/appfx/`)
```go
package appfx

var Module = fx.Options(
    configfx.Module,
    parserfx.Module,
    embeddingsfx.Module,
    storagefx.Module,
    searchfx.Module,
    indexerfx.Module,
    mcpfx.Module,
    cmdsfx.Module,
)
```
- 组合所有模块
- 提供应用程序工厂方法

## 🎯 设计原则

### 1. **单一职责原则**
每个 fx 包只负责一个具体的功能域：
- `configfx` - 配置管理
- `parserfx` - 代码解析
- `embeddingsfx` - 向量嵌入
- `storagefx` - 数据存储
- 等等...

### 2. **清晰的依赖关系**
```
configfx (基础层)
    ↓
parserfx, embeddingsfx, storagefx (服务层)
    ↓
searchfx, indexerfx (业务层)
    ↓
mcpfx, cmdsfx (应用层)
    ↓
appfx (组装层)
```

### 3. **接口分离**
每个模块通过明确定义的接口进行交互，避免紧耦合。

### 4. **可测试性**
每个模块都可以独立测试，支持模拟和存根。

## 📦 使用示例

### 创建应用程序
```go
import "github.com/0x5457/ts-index/internal/app/appfx"

// 使用配置创建应用
app := appfx.NewAppWithConfig(
    dbPath,
    embedURL,
    project,
)

// 或使用默认配置
app := appfx.NewApp()
```

### 使用特定模块
```go
import (
    "github.com/0x5457/ts-index/internal/config/configfx"
    "github.com/0x5457/ts-index/internal/parser/parserfx"
)

app := fx.New(
    configfx.Module,
    parserfx.Module,
    fx.Invoke(func(config *configfx.Config, parser parser.Parser) {
        // 使用配置和解析器
    }),
)
```

### 命令集成
```go
import (
    "github.com/0x5457/ts-index/cmd/cmdsfx"
    "github.com/0x5457/ts-index/internal/app/appfx"
)

app := fx.New(
    appfx.Module,
    fx.Supply(
        fx.Annotate(dbPath, fx.ResultTags(`name:"dbPath"`)),
        fx.Annotate(embedURL, fx.ResultTags(`name:"embedURL"`)),
        fx.Annotate(project, fx.ResultTags(`name:"project"`)),
    ),
    fx.Invoke(func(runner *cmdsfx.CommandRunner) error {
        return runner.RunIndex(ctx, projectPath)
    }),
)
```

## 🧪 测试策略

### 模块级测试
每个模块都有自己的测试文件：
- `configfx/module_test.go` - 配置模块测试
- `parserfx/module_test.go` - 解析器模块测试
- `appfx/module_test.go` - 应用模块集成测试

### 集成测试
通过 `appfx` 模块进行完整的集成测试，确保所有模块正确协作。

## 🔧 开发工作流

### 1. 添加新模块
```bash
# 创建新模块目录
mkdir -p internal/newfeature/newfeaturefx

# 创建模块文件
cat > internal/newfeature/newfeaturefx/module.go << EOF
package newfeaturefx

import "go.uber.org/fx"

// 模块定义
var Module = fx.Module("newfeature",
    fx.Provide(NewService),
)
EOF
```

### 2. 集成到应用
在 `appfx/module.go` 中添加新模块：
```go
var Module = fx.Options(
    // ... 现有模块
    newfeaturefx.Module,
)
```

### 3. 质量检查
```bash
make lint-fix  # 代码格式化和 lint
make test      # 运行测试
go build ./... # 编译检查
```

## 📈 优势

### 1. **可维护性**
- 每个模块职责清晰
- 模块间依赖关系明确
- 易于定位和修复问题

### 2. **可扩展性**
- 添加新功能无需修改现有模块
- 模块可以独立演进
- 支持渐进式重构

### 3. **可测试性**
- 每个模块可以独立测试
- 支持依赖注入和模拟
- 测试更加精确和快速

### 4. **团队协作**
- 不同团队可以并行开发不同模块
- 清晰的模块边界减少冲突
- 代码审查更加聚焦

## 🚀 最佳实践

### 1. **模块设计**
- 保持模块小而专注
- 明确定义模块边界
- 使用接口进行模块间通信

### 2. **依赖管理**
- 避免循环依赖
- 优先使用接口而非具体类型
- 使用 `fx.In` 结构体组织参数

### 3. **错误处理**
- 在模块边界进行错误包装
- 提供有意义的错误信息
- 支持错误链追踪

### 4. **生命周期管理**
- 正确使用 `fx.Lifecycle` 钩子
- 确保资源的正确清理
- 支持优雅关闭

## 📊 质量保证

✅ **编译通过** - `go build ./...`  
✅ **Lint 检查** - `make lint-fix` (0 issues)  
✅ **测试通过** - `make test` (所有测试通过)  
✅ **模块化设计** - 每个功能都有独立的 fx 包  
✅ **依赖注入** - 使用 Uber Fx 最佳实践  
✅ **文档完整** - 全面的架构文档和使用示例

这种模块化架构为项目的长期维护和发展奠定了坚实的基础。