package treesitterutil

import (
	"context"
	"fmt"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

// ParserPool reuses *sitter.Parser instances for one language.
//
// Tree-sitter parsers are not safe for concurrent use, but constructing one
// is expensive (it allocates C memory through CGo and sets the language).
// A sync.Pool of warmed-up parsers, one pool per language, is the idiomatic
// solution.
//
// Go note: sync.Pool returns objects via its New func when empty. We can't
// use a plain `[]*sitter.Parser` because that would require manual locking
// and wouldn't play nicely with the GC's scavenger.
type ParserPool struct {
	lang *sitter.Language
	pool sync.Pool
}

// NewParserPool returns an empty pool for lang. Parsers are created lazily
// on the first Get().
func NewParserPool(lang *sitter.Language) *ParserPool {
	return &ParserPool{
		lang: lang,
		pool: sync.Pool{
			New: func() any {
				parser := sitter.NewParser()
				parser.SetLanguage(lang)
				return parser
			},
		},
	}
}

// Get borrows a parser from the pool. Always return it with Put(), usually
// via defer:
//
//	p := pool.Get()
//	defer pool.Put(p)
//
// Go note: sync.Pool.Get returns `any`, so we type-assert to the concrete
// type with `.(*sitter.Parser)`. The single-return form panics on a type
// mismatch, which is fine here because only our New callback ever puts
// values into this pool.
func (pp *ParserPool) Get() *sitter.Parser {
	return pp.pool.Get().(*sitter.Parser)
}

// Put returns a parser to the pool so it can be reused by a later Get().
//
// Callers must not pass nil: sync.Pool only treats a nil *interface* as
// no-op, not a typed nil pointer wrapped in `any`. Passing nil here would
// poison the pool and later cause a nil-pointer panic inside Parse.
func (pp *ParserPool) Put(p *sitter.Parser) {
	if p == nil {
		return
	}
	pp.pool.Put(p)
}

// Parse is a one-shot helper: borrow a parser, parse source, release the
// parser, return the resulting tree. The caller owns the returned tree
// and must Close() it when finished.
//
// ctx is honoured: cancelling ctx aborts an in-flight parse and ParseCtx
// returns a non-nil error.
//
// Go note: `defer pp.Put(parser)` guarantees the parser goes back to the
// pool on every exit path — normal return, error return, or panic. It is
// Go's idiomatic stand-in for try/finally.
func (pp *ParserPool) Parse(ctx context.Context, source []byte) (*sitter.Tree, error) {
	parser := pp.Get()

	tree, err := parser.ParseCtx(ctx, nil, source)
	if err != nil {
		return nil, fmt.Errorf("treesitterutil: parse: %w", err)
	}
	// only return valid parsers into the pool
	defer pp.Put(parser)
	return tree, nil
}

// Language returns the grammar this pool was constructed with.
func (pp *ParserPool) Language() *sitter.Language {
	return pp.lang
}
