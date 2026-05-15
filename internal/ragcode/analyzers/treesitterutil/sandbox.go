package treesitterutil

// Reference / scratch code that predates the public API. It hard-codes the C
// grammar to verify the smacker/go-tree-sitter wiring works end-to-end.
//
// Once ParserPool.Parse is implemented this file should be deleted.

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
)

// parseCCode parses a C snippet using a fresh parser. Not for production use —
// it allocates a new parser per call and never returns it to a pool.
func parseCCode(code string) (*sitter.Tree, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(c.GetLanguage())
	return parser.ParseCtx(context.Background(), nil, []byte(code))
}
