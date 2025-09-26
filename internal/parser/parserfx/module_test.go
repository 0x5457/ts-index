package parserfx

import (
	"context"
	"testing"

	"github.com/0x5457/ts-index/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestParserModule(t *testing.T) {
	var parser parser.Parser
	app := fx.New(
		Module,
		fx.Populate(&parser),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	defer func() {
		require.NoError(t, app.Stop(ctx))
	}()

	assert.NotNil(t, parser)
}
