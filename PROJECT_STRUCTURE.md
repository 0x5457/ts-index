# 项目模块化结构概览

本项目已完成系统性重构，采用真正的模块化Fx依赖注入架构。

## 🏗️ 目录结构

```
ts-index/
├── cmd/
│   ├── cmdsfx/                    # 命令执行模块
│   │   └── module.go             # CommandRunner 实现
│   └── ts-index/
│       ├── commands/             # CLI 命令定义
│       └── main.go              # 应用入口点
│
├── internal/
│   ├── app/
│   │   └── appfx/               # 应用组装模块
│   │       ├── module.go        # 组合所有模块
│   │       └── module_test.go   # 集成测试
│   │
│   ├── config/
│   │   └── configfx/            # 配置管理模块
│   │       ├── module.go        # 配置提供者
│   │       └── module_test.go   # 配置测试
│   │
│   ├── parser/
│   │   ├── parserfx/            # 解析器模块
│   │   │   ├── module.go        # 解析器提供者
│   │   │   └── module_test.go   # 解析器测试
│   │   ├── tsparser/            # TypeScript解析实现
│   │   └── parser.go            # 解析器接口
│   │
│   ├── embeddings/
│   │   ├── embeddingsfx/        # 嵌入模块
│   │   │   └── module.go        # 嵌入服务提供者
│   │   ├── api.go               # API嵌入实现
│   │   ├── local.go             # 本地嵌入实现
│   │   └── embedder.go          # 嵌入器接口
│   │
│   ├── storage/
│   │   ├── storagefx/           # 存储模块
│   │   │   └── module.go        # 存储服务提供者
│   │   ├── sqlite/              # SQLite符号存储
│   │   ├── sqlvec/              # SQLite向量存储
│   │   └── storage.go           # 存储接口
│   │
│   ├── search/
│   │   ├── searchfx/            # 搜索模块
│   │   │   └── module.go        # 搜索服务提供者
│   │   └── service.go           # 搜索服务实现
│   │
│   ├── indexer/
│   │   ├── indexerfx/           # 索引器模块
│   │   │   └── module.go        # 索引器提供者
│   │   ├── pipeline/            # 索引管道实现
│   │   └── indexer.go           # 索引器接口
│   │
│   ├── mcp/
│   │   ├── mcpfx/               # MCP模块
│   │   │   └── module.go        # MCP服务器提供者
│   │   ├── client.go            # MCP客户端
│   │   └── server.go            # MCP服务器实现
│   │
│   ├── lsp/                     # LSP支持
│   ├── models/                  # 数据模型
│   ├── constants/               # 常量定义
│   └── util/                    # 工具函数
│
├── examples/                    # 示例项目
├── scripts/                     # 构建脚本
├── Makefile                     # 构建配置
├── go.mod                       # Go模块定义
├── FX_MODULAR_ARCHITECTURE.md  # 详细架构文档
└── PROJECT_STRUCTURE.md        # 本文档
```

## 🎯 模块化设计原则

### 1. **独立的Fx包**
每个功能域都有自己的 `*fx` 包：
- `configfx` - 配置管理
- `parserfx` - 代码解析  
- `embeddingsfx` - 向量嵌入
- `storagefx` - 数据存储
- `searchfx` - 语义搜索
- `indexerfx` - 代码索引
- `mcpfx` - MCP服务器
- `cmdsfx` - 命令执行
- `appfx` - 应用组装

### 2. **清晰的依赖层次**
```
配置层 (configfx)
    ↓
基础服务层 (parserfx, embeddingsfx, storagefx)
    ↓
业务逻辑层 (searchfx, indexerfx)
    ↓
应用服务层 (mcpfx, cmdsfx)
    ↓
组装层 (appfx)
```

### 3. **测试策略**
- 每个模块都有独立的测试
- 集成测试在 `appfx` 层进行
- E2E测试使用专门的测试配置

## 🔧 开发工作流

### 添加新模块
1. 创建 `internal/newfeature/newfeaturefx/` 目录
2. 实现 `module.go` 和 `module_test.go`
3. 在 `appfx/module.go` 中集成
4. 运行测试和lint检查

### 质量保证
```bash
make lint-fix  # ✅ 0 issues
make test      # ✅ 所有测试通过
go build ./... # ✅ 编译成功
```

## 🚀 使用示例

### 基本应用创建
```go
import "github.com/0x5457/ts-index/internal/app/appfx"

app := appfx.NewAppWithConfig(dbPath, embedURL, project)
```

### 自定义模块组合
```go
import (
    "github.com/0x5457/ts-index/internal/config/configfx"
    "github.com/0x5457/ts-index/internal/parser/parserfx"
)

app := fx.New(
    configfx.Module,
    parserfx.Module,
    // ... 其他需要的模块
)
```

### 命令执行
```go
import "github.com/0x5457/ts-index/cmd/cmdsfx"

app := fx.New(
    appfx.Module,
    fx.Supply(/* 配置参数 */),
    fx.Invoke(func(runner *cmdsfx.CommandRunner) error {
        return runner.RunIndex(ctx, projectPath)
    }),
)
```

## 📊 优势总结

✅ **模块化** - 每个功能有独立的fx包  
✅ **可测试** - 完整的测试覆盖  
✅ **可维护** - 清晰的依赖关系  
✅ **可扩展** - 易于添加新功能  
✅ **质量保证** - lint和test全部通过  
✅ **文档完整** - 详细的架构说明  

这种架构为项目的长期发展提供了坚实的基础。