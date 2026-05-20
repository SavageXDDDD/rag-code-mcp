package treesitterutil

import (
	"slices"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CleanLineComment strips marker from the start of s and returns the
// trimmed body. The marker is supplied by the caller so the same helper
// works across languages — pass "//" for C/Java/Go, "#" for shell/Python,
// "///" to collapse Doxygen-style markers. Any string prefix works; if
// it doesn't match, s is returned with whitespace trimmed.
//
//	CleanLineComment("// hello",     "//")  == "hello"
//	CleanLineComment("#  hello",     "#")   == "hello"
//	CleanLineComment("/// doxy",     "///") == "doxy"
//	CleanLineComment("plain text",   "//")  == "plain text"
func CleanLineComment(s, marker string) string {
	ret, _ := strings.CutPrefix(s, marker)
	return strings.TrimSpace(ret)
}

// Recognised comment markers, used by LeadingDocComment to decide how to
// clean a comment node's raw text. singleLineCommentMarkers is scanned
// longest-first so "///" matches before "//" — preserve that invariant if
// you add a new marker (e.g. "//!" for Doxygen, "--" for Lua).
var (
	singleLineCommentMarkers = []string{"///", "//", "#"}
	multilineCommentMarkers  = []string{"/*"}
)

// CleanBlockComment normalises a /* ... */ (or /** ... */) comment body
// into a clean docstring. The transformation is line-by-line:
//
//   - Strips the leading "/*" / "/**" opener and the trailing "*/" closer.
//
//   - On each interior line: trims whitespace, then if the line begins
//     with "* " (the Javadoc/Doxygen continuation marker) strips that
//     marker too. Lines that are exactly "*" are dropped entirely.
//
//   - Lines beginning with "**" are left as-is, so Markdown bold at the
//     start of a doc line ("**Important**: …") survives.
//
//   - Empty lines are dropped, which means paragraph breaks inside the
//     comment collapse. Acceptable for vector-search indexing; if you need
//     to render the docstring verbatim, pass the raw bytes through.
//
//     in:  "/**\n * Hello.\n * World.\n */"
//     out: "Hello.\nWorld."
//
//     in:  "/* foo */"
//     out: "foo"
//
//     in:  "/**\n * **Important**: see foo().\n */"
//     out: "**Important**: see foo()."
func CleanBlockComment(s string) string {
	const multilineCommentPrefix = "/*"
	const multilineCommentSuffix = "*/"
	cut, _ := strings.CutPrefix(s, multilineCommentPrefix)
	cut, _ = strings.CutSuffix(cut, multilineCommentSuffix)

	split := strings.Split(cut, "\n")
	goodLines := make([]string, 0, len(split))
	for _, line := range split {
		line = strings.TrimSpace(line)
		if line == "*" {
			continue
		} else if strings.HasPrefix(line, "* ") {
			line = strings.TrimSpace(line[2:])
		}
		if line != "" {
			goodLines = append(goodLines, line)
		}
	}

	return strings.Join(goodLines, "\n")
}

// LeadingDocComment looks at the immediately-preceding named sibling of
// n and, if it is a comment node, returns its cleaned text. Returns ""
// when n is nil, has no preceding sibling, the sibling is not a comment,
// or the comment node spans no bytes.
//
// commentTypes is the set of grammar-specific node-type strings that
// should be treated as comments. The variadic form keeps the helper
// grammar-agnostic — typical calls:
//
//	LeadingDocComment(n, src, "comment")                   // C, C++, Go
//	LeadingDocComment(n, src, "line_comment", "block_comment") // Java
//	LeadingDocComment(n, src, "line_comment", "multiline_comment") // Kotlin
//
// Cleaning is dispatched by the comment's prefix:
//   - "/*" → CleanBlockComment
//   - otherwise, matched against singleLineCommentMarkers (longest first:
//     "///", "//", "#") and stripped via CleanLineComment with the
//     matching marker.
//
// Known limitations:
//   - Only the immediately-adjacent sibling is examined. If an annotation
//     (Java @Deprecated, Kotlin @JvmStatic) or attribute sits between the
//     doc comment and the declaration, the comment is missed. Extractors
//     that care can walk back past modifier/annotation nodes themselves.
//   - Multiple consecutive single-line comments ("// a\n// b") are usually
//     emitted as separate sibling nodes; this helper returns only the
//     last one. Collapsing a run is the caller's job.
//   - Python is not handled — its docstrings live inside the function body
//     as the first statement, not as a leading sibling.
func LeadingDocComment(n *sitter.Node, source []byte, commentTypes ...string) string {
	if n == nil {
		return ""
	}

	sib := n.PrevNamedSibling()
	if sib == nil {
		return ""
	}

	if slices.Contains(commentTypes, sib.Type()) {
		content := sib.Content(source)
		if len(content) == 0 {
			return ""
		}

		for _, marker := range multilineCommentMarkers {
			if strings.HasPrefix(content, marker) {
				return CleanBlockComment(content)
			}
		}

		// TODO: Add multiple single line comment handling & merging
		for _, marker := range singleLineCommentMarkers {
			if strings.HasPrefix(content, marker) {
				return CleanLineComment(content, marker)
			}
		}
	}

	return ""
}
