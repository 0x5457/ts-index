package fx

import (
	"github.com/0x5457/ts-index/internal/parser"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"go.uber.org/fx"
)

// NewParser creates a new TypeScript parser instance
func NewParser() parser.Parser {
	return tsparser.New()
}

// ParserModule provides parser components
var ParserModule = fx.Module("parser",
	fx.Provide(NewParser),
)
