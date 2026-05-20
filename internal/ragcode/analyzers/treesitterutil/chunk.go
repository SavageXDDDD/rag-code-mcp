package treesitterutil

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/doITmagic/rag-code-mcp/internal/codetypes"
)

// FillLocation populates the location-and-source fields of c from a
// tree-sitter node — FilePath, StartLine, EndLine, Code. The
// symbol-level fields (Name, Type, Package, Signature, Docstring, ...)
// remain the caller's responsibility.
//
// If n is nil the function is a no-op; c is left untouched, FilePath
// included. Typical call site inside an Extractor:
//
//	chunk := codetypes.CodeChunk{
//	    Type:     "function",
//	    Name:     NodeText(nameNode, source),
//	    Language: "c",
//	}
//	FillLocation(&chunk, n, source, path)
//	FillSelection(&chunk, nameNode)
//
// Go note: c is *codetypes.CodeChunk, so writes mutate the caller's
// chunk in place. Returning the struct by value would force the caller
// to reassign and would discard any fields the caller had already set.
func FillLocation(c *codetypes.CodeChunk, n *sitter.Node, source []byte, filePath string) {
	if n == nil {
		return
	}
	c.FilePath = filePath
	c.StartLine = StartLine(n)
	c.EndLine = EndLine(n)
	c.Code = n.Content(source)
}

// FillSelection sets SelectionStartLine/EndLine on c to the row range of
// nameNode — the node that holds the symbol's identifier. This is what an
// IDE jumps to when the user "goes to definition": not the whole
// declaration, just the name itself.
//
// nameNode is grammar-specific. Common ways to obtain it:
//
//	nameNode := n.ChildByFieldName("name")        // most grammars
//	nameNode := FindChildByType(n, "identifier")  // fallback
//
// If nameNode is nil the function is a no-op, leaving both selection
// fields at their zero value.
func FillSelection(c *codetypes.CodeChunk, nameNode *sitter.Node) {
	if nameNode == nil {
		return
	}
	c.SelectionStartLine = StartLine(nameNode)
	c.SelectionEndLine = EndLine(nameNode)
}

// ExtractSignatureLine returns the first source line of n, trimmed of
// surrounding whitespace. It's a fallback for CodeChunk.Signature when
// the grammar doesn't expose a dedicated signature subtree — Kotlin
// top-level properties, C variable declarations, etc.
//
// Prefer a real signature node when one exists (Java
// "method_declaration"'s explicit children, Go's parameter_list, ...).
// This helper is intentionally crude: it does not understand multi-line
// declarations, so a C function declared as
//
//	int really_long_function_name(
//	    int a, int b);
//
// will yield "int really_long_function_name(" — the parameter context is
// lost. Use it where one-line declarations are the norm.
//
// Returns "" if n is nil or spans no bytes.
func ExtractSignatureLine(n *sitter.Node, source []byte) string {
	if n == nil {
		return ""
	}
	contents := n.Content(source)

	if i := strings.IndexByte(contents, '\n'); i >= 0 {
		contents = contents[:i]
	}
	return strings.TrimSpace(contents)
}
