// Package treesitterutil bundles the parser pool, node-traversal helpers,
// comment-cleaning utilities and CodeChunk builders shared by every
// tree-sitter–backed language analyzer (Java, Kotlin, C, C++, ...).
//
// A language analyzer wires it up like this:
//
//	type Analyzer struct{ *treesitterutil.CodeAnalyzer }
//
//	func NewAnalyzer() *Analyzer {
//	    return &Analyzer{
//	        CodeAnalyzer: treesitterutil.NewCodeAnalyzer(treesitterutil.Options{
//	            Language:   java.GetLanguage(),
//	            Extensions: []string{".java"},
//	            Extract:    extractJavaChunks, // user-supplied
//	        }),
//	    }
//	}
//
// The user-supplied Extractor walks the syntax tree and emits one
// codetypes.CodeChunk per symbol. Everything else (parser reuse, file
// walking, line numbers, comment normalisation) is handled here.
package treesitterutil

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/doITmagic/rag-code-mcp/internal/codetypes"
)

// Extractor converts a parsed syntax tree into CodeChunks. Each language
// analyzer supplies one.
//
// Go note: function types are first-class values. Storing the extractor as
// a field on Options means callers can swap parsing strategies without
// inheritance.
type Extractor func(root *sitter.Node, source []byte, path string) ([]codetypes.CodeChunk, error)

// Options configures a CodeAnalyzer. The zero value is not useful — Language
// and Extract must be set.
type Options struct {
	// Language is the tree-sitter grammar to parse with.
	Language *sitter.Language

	// Extensions is the set of file suffixes to index, e.g. {".java"} or
	// {".cpp", ".cc", ".hpp"}. Match is case-sensitive.
	Extensions []string

	// SkipDirs lists directory base names to prune while walking. If nil,
	// DefaultSkipDirs() is used.
	SkipDirs []string

	// SkipTests, when true, omits common test-file patterns from the walk.
	SkipTests bool

	// Extract is called once per parsed file. Required.
	Extract Extractor
}

type TreeSitterParser interface {
	GetNextChunk() codetypes.CodeChunk
}

// CodeAnalyzer is the generic tree-sitter-backed PathAnalyzer. Per-language
// analyzers embed *CodeAnalyzer and supply an Extractor via Options.
type CodeAnalyzer struct {
	opts Options
	pool *ParserPool
}

// NewCodeAnalyzer returns a CodeAnalyzer using DefaultSkipDirs when
// opts.SkipDirs is nil. Use NewCodeAnalyzerWithOptions to bypass that
// default.
func NewCodeAnalyzer(opts Options) *CodeAnalyzer {
	if opts.SkipDirs == nil {
		opts.SkipDirs = DefaultSkipDirs()
	}
	return NewCodeAnalyzerWithOptions(opts)
}

// NewCodeAnalyzerWithOptions returns a CodeAnalyzer with all fields under
// caller control. Callers must set opts.Language and opts.Extract.
func NewCodeAnalyzerWithOptions(opts Options) *CodeAnalyzer {
	panic("TODO: implement (allocate ParserPool, store opts, return &CodeAnalyzer{...})")
}

// AnalyzePaths satisfies codetypes.PathAnalyzer: it walks each input path,
// parses every matching file, runs the Extractor, and returns the union of
// all chunks. Per-file errors should be logged to stderr but not abort the
// walk (mirroring the other analyzers).
func (ca *CodeAnalyzer) AnalyzePaths(paths []string) ([]codetypes.CodeChunk, error) {
	panic("TODO: implement (call WalkSourceFiles, then ParseAndExtract per file)")
}

// AnalyzeFile parses a single file and returns its CodeChunks.
func (ca *CodeAnalyzer) AnalyzeFile(filePath string) ([]codetypes.CodeChunk, error) {
	panic("TODO: implement (os.ReadFile + ParseAndExtract)")
}

// ParseAndExtract is the lower-level entry point: parse source, hand the
// root node to the Extractor, return the chunks. Useful when the caller
// already has the bytes in memory (e.g. unit tests).
func (ca *CodeAnalyzer) ParseAndExtract(filePath string, source []byte) ([]codetypes.CodeChunk, error) {
	panic("TODO: implement (ca.pool.Parse + ca.opts.Extract)")
}

// Pool exposes the underlying ParserPool for advanced callers that need to
// drive the parser themselves (e.g. incremental reparsing).
func (ca *CodeAnalyzer) Pool() *ParserPool {
	return ca.pool
}

// DefaultSkipDirs returns directory base names that are almost never worth
// indexing. A fresh slice is returned each call, so callers may mutate it
// freely.
func DefaultSkipDirs() []string {
	return []string{
		".git", ".hg", ".svn",
		"node_modules", "vendor",
		"build", "dist", "out", "target", "bin", "obj",
		".gradle", ".idea", ".vscode", ".cache",
	}
}
