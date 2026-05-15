package treesitterutil

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// NodeText returns the source bytes the node spans, decoded as a string.
// Returns "" for a nil node.
func NodeText(n *sitter.Node, source []byte) string {
	panic("TODO: implement — guard nil, then n.Content(source)")
}

// StartLine returns the 1-based start line of n (0 for nil).
//
// Tree-sitter exposes 0-based row numbers; CodeChunk uses 1-based. Always
// go through StartLine/EndLine — never expose raw .Row to callers.
func StartLine(n *sitter.Node) int {
	panic("TODO: implement")
}

// EndLine returns the 1-based end line of n (0 for nil).
func EndLine(n *sitter.Node) int {
	panic("TODO: implement")
}

// FindChildByType returns the first direct named child of n whose Type()
// matches t, or nil if none match.
func FindChildByType(n *sitter.Node, t string) *sitter.Node {
	panic("TODO: implement")
}

// FindChildrenByType returns all direct named children of n whose Type()
// matches t, in source order.
func FindChildrenByType(n *sitter.Node, t string) []*sitter.Node {
	panic("TODO: implement")
}

// WalkNamed performs a depth-first walk over all named descendants of n
// (including n itself), invoking fn on each. Return false from fn to skip
// descent into that node's children; return true to continue.
//
// Go note: returning a bool from the callback is a common Go idiom for
// "should we keep walking?" — see filepath.WalkDir's fs.SkipDir for a
// similar pattern with sentinel errors.
func WalkNamed(n *sitter.Node, fn func(*sitter.Node) bool) {
	panic("TODO: implement")
}

// FindFirstNamed returns the first descendant (DFS, named-only) for which
// pred returns true, or nil if none match.
func FindFirstNamed(n *sitter.Node, pred func(*sitter.Node) bool) *sitter.Node {
	panic("TODO: implement")
}
