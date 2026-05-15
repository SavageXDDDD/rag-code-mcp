package treesitterutil

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// CleanLineComment strips a single-line comment marker (// or #) and
// surrounding whitespace. The marker is supplied so the same helper works
// across languages.
//
//	CleanLineComment("// hello",  "//") == "hello"
//	CleanLineComment("#  hello",  "#")  == "hello"
func CleanLineComment(s, marker string) string {
	panic("TODO: implement")
}

// CleanBlockComment removes /* ... */ and /** ... */ wrappers and the
// leading-star convention used by Javadoc/Doxygen, returning a clean
// multi-line docstring with no trailing blank lines.
//
//	in:  "/**\n * Hello.\n * World.\n */"
//	out: "Hello.\nWorld."
func CleanBlockComment(s string) string {
	panic("TODO: implement")
}

// LeadingDocComment walks the named siblings preceding n and returns the
// cleaned text of an immediately-adjacent comment node (if its Type() is
// one of commentTypes). Returns "" if there is no leading comment.
//
// The variadic commentTypes lets callers pass the grammar-specific node
// names — Java uses "block_comment"/"line_comment", C uses "comment", etc.
func LeadingDocComment(n *sitter.Node, source []byte, commentTypes ...string) string {
	panic("TODO: implement")
}
