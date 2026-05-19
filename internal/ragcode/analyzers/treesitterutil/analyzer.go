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
	"context"
	"fmt"
	"os"

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

// CodeAnalyzer is the generic tree-sitter-backed PathAnalyzer. Per-language
// analyzers embed *CodeAnalyzer and supply an Extractor via Options.
type CodeAnalyzer struct {
	opts Options
	pool *ParserPool
}

// NewCodeAnalyzer is the standard constructor: it fills in DefaultSkipDirs
// when opts.SkipDirs is nil and otherwise delegates to
// NewCodeAnalyzerWithOptions. Use this in all call sites that want the
// project-wide skip defaults; use NewCodeAnalyzerWithOptions directly if
// you need a literally empty skip list.
func NewCodeAnalyzer(opts Options) *CodeAnalyzer {
	if opts.SkipDirs == nil {
		opts.SkipDirs = DefaultSkipDirs()
	}
	return NewCodeAnalyzerWithOptions(opts)
}

// NewCodeAnalyzerWithOptions returns a CodeAnalyzer with all fields under
// caller control — no defaults are applied. Callers must set opts.Language
// and opts.Extract; opts.Extensions must be non-empty if AnalyzePaths is
// going to find anything. The ParserPool is allocated eagerly so callers
// can rely on Pool() being non-nil from this point on.
func NewCodeAnalyzerWithOptions(opts Options) *CodeAnalyzer {
	return &CodeAnalyzer{
		opts: opts,
		pool: NewParserPool(opts.Language),
	}
}

// AnalyzePaths satisfies codetypes.PathAnalyzer. It walks each input path
// via WalkSourceFiles, parses every matching file, runs the Extractor, and
// returns the union of all chunks.
//
// Error policy: per-file failures (read, parse, extract) are reported via
// the stderr warning printed by ParseAndExtract and silently dropped here
// — the walk continues to the next file. The returned error is non-nil
// only when WalkSourceFiles itself fails (e.g. a directory could not be
// listed because of permissions); in that case the walk has aborted and
// the returned chunk slice is nil.
//
// This mirrors how the Python/PHP analyzers in the codebase handle bulk
// indexing: one broken file does not lose the entire run.
func (ca *CodeAnalyzer) AnalyzePaths(paths []string) ([]codetypes.CodeChunk, error) {
	var out []codetypes.CodeChunk

	err := WalkSourceFiles(paths, ca.opts.Extensions, ca.opts.SkipDirs, ca.opts.SkipTests, func(path string, content []byte) error {
		chunks, _ := ca.ParseAndExtract(path, content)
		out = append(out, chunks...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("treesitterutil: AnalyzePaths: %w", err)
	}

	return out, nil
}

// AnalyzeFile reads filePath from disk and returns its CodeChunks. Unlike
// AnalyzePaths it does not swallow errors — read failures and parse
// failures are wrapped with %w and returned to the caller, so
// errors.Is/As against os.ErrNotExist or the smacker sentinels still
// works.
//
// Use this when you have a single, known file and want the caller to
// decide how to react to failure. For bulk indexing use AnalyzePaths.
func (ca *CodeAnalyzer) AnalyzeFile(filePath string) ([]codetypes.CodeChunk, error) {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("treesitterutil: AnalyzeFile: %w", err)
	}

	return ca.ParseAndExtract(filePath, contents)
}

// ParseAndExtract is the lower-level entry point used by AnalyzePaths and
// AnalyzeFile. It borrows a parser from the pool, parses source, defers
// closing the resulting *sitter.Tree (the tree owns C memory and must be
// released explicitly, not relied on the GC finalizer), and hands the
// root node to the Extractor.
//
// On parse failure it logs a single warning line to stderr and returns
// the parse error wrapped with %w. filePath is only used for diagnostics —
// no I/O is performed by this method, so callers that already have the
// bytes in memory (unit tests, in-memory caches, the AnalyzePaths
// visitor) can drive parsing without round-tripping through the
// filesystem.
func (ca *CodeAnalyzer) ParseAndExtract(filePath string, source []byte) ([]codetypes.CodeChunk, error) {
	tree, err := ca.pool.Parse(context.TODO(), source)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "treesitterutil: parse: %v\n", err)
		return nil, fmt.Errorf("treesitterutil: ParseAndExtract: %w", err)
	}
	defer tree.Close()
	return ca.opts.Extract(tree.RootNode(), source, filePath)

}

// Pool exposes the underlying ParserPool. Most callers should not need
// this — it exists for advanced uses that bypass the Extractor pipeline,
// such as incremental reparsing of a single file as the user types, or
// running tree-sitter queries directly against a node tree. The returned
// pool is shared with this CodeAnalyzer; concurrent Get/Put is safe but
// callers must Put every parser they Get.
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
