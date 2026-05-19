package treesitterutil

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// NodeText returns the source bytes the node spans, decoded as a string.
// Returns "" for a nil node.
func NodeText(n *sitter.Node, source []byte) string {
	if n == nil {
		return ""
	}
	return n.Content(source)
}

// StartLine returns the 1-based start line of n (0 for nil).
//
// Tree-sitter exposes 0-based row numbers; CodeChunk uses 1-based. Always
// go through StartLine/EndLine — never expose raw .Row to callers.
func StartLine(n *sitter.Node) int {
	if n == nil {
		return 0
	}
	return int(n.StartPoint().Row + 1)
}

// EndLine returns the 1-based end line of n (0 for nil).
func EndLine(n *sitter.Node) int {
	if n == nil {
		return 0
	}
	return int(n.EndPoint().Row + 1)
}

// FindChildByType returns the first direct named child of n whose Type()
// matches t, or nil if none match.
func FindChildByType(n *sitter.Node, t string) *sitter.Node {
	if n == nil {
		return nil
	}

	for i := range int(n.NamedChildCount()) {
		child := n.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() == t {
			return child
		}
	}

	return nil
}

// FindChildrenByType returns all direct named children of n whose Type()
// matches t, in source order.
func FindChildrenByType(n *sitter.Node, t string) []*sitter.Node {
	if n == nil {
		return nil
	}

	var children []*sitter.Node

	for i := range int(n.NamedChildCount()) {
		child := n.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() == t {
			children = append(children, child)
		}
	}

	return children
}

// WalkNamed performs a depth-first walk over all named descendants of n
// (including n itself), invoking fn on each. Return false from fn to skip
// descent into that node's children; return true to continue.
//
// Go note: returning a bool from the callback is a common Go idiom for
// "should we keep walking?" — see filepath.WalkDir's fs.SkipDir for a
// similar pattern with sentinel errors.
func WalkNamed(n *sitter.Node, fn func(*sitter.Node) bool) {
	if n == nil {
		return
	}

	if fn(n) {
		for i := range int(n.NamedChildCount()) {
			child := n.NamedChild(i)
			if child == nil {
				continue
			}

			WalkNamed(child, fn)
		}
	}
}

// FindFirstNamed returns the first descendant (DFS, named-only) for which
// pred returns true, or nil if none match.
func FindFirstNamed(n *sitter.Node, pred func(*sitter.Node) bool) *sitter.Node {
	if n == nil {
		return nil
	}

	for i := range int(n.NamedChildCount()) {
		child := n.NamedChild(i)
		if child == nil {
			continue
		}

		if pred(child) {
			return child
		}

		found := FindFirstNamed(child, pred)
		if found != nil {
			return found
		}
	}

	return nil
}
