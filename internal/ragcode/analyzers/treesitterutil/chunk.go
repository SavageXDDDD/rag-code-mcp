package treesitterutil

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/doITmagic/rag-code-mcp/internal/codetypes"
)

// FillLocation sets the FilePath / StartLine / EndLine / Code fields of c
// from a tree-sitter node. The caller is still responsible for the
// symbol-level fields (Name, Type, Package, Signature, Docstring, ...).
//
// Go note: c is *codetypes.CodeChunk, so writes mutate the caller's chunk.
// Returning the struct by value would force the caller to reassign.
func FillLocation(c *codetypes.CodeChunk, n *sitter.Node, source []byte, filePath string) {
	panic("TODO: implement")
}

// FillSelection sets SelectionStartLine/EndLine to the range of the symbol
// name node (e.g. the identifier inside a function declaration).
func FillSelection(c *codetypes.CodeChunk, nameNode *sitter.Node) {
	panic("TODO: implement")
}

// ExtractSignatureLine returns the first source line of the node, trimmed.
// Useful as a fallback Signature when there's no dedicated signature node
// in the grammar (e.g. Kotlin top-level properties).
func ExtractSignatureLine(n *sitter.Node, source []byte) string {
	panic("TODO: implement")
}
