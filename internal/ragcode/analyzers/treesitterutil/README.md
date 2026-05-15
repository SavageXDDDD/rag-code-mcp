# treesitterutil

Shared scaffolding for tree-sitter–backed language analyzers (Java, Kotlin,
C, C++, ...). It is **not** a `PathAnalyzer` itself — it gives each language
analyzer a parser pool, a file walker, node helpers, and CodeChunk builders
so they can stay small and focused on grammar-specific extraction.

This package is **Phase 0** in `../../../../../Plan.md`.

## Status

All public functions are declared with `panic("TODO: implement")` bodies.
The package compiles; runtime calls into the stubs panic. Implement them
incrementally — start with `ParserPool`, then `WalkSourceFiles`, then the
node and comment helpers.

`sandbox.go` keeps the original `parseCCode` reference snippet for now;
delete it once `ParserPool.Parse` works.

## Files

| File          | Contents                                                                |
| ------------- | ----------------------------------------------------------------------- |
| `analyzer.go` | Package doc, `Options`, `Extractor`, `CodeAnalyzer`, `DefaultSkipDirs`. |
| `parser.go`   | `ParserPool` (`sync.Pool` of `*sitter.Parser`) + one-shot `Parse`.      |
| `node.go`     | `NodeText`, `StartLine`/`EndLine`, `FindChild(ren)ByType`, `WalkNamed`. |
| `chunk.go`    | `FillLocation`, `FillSelection`, `ExtractSignatureLine`.                |
| `comment.go`  | `CleanLineComment`, `CleanBlockComment`, `LeadingDocComment`.           |
| `walk.go`     | `WalkSourceFiles` — filesystem walker with skip-dirs / skip-tests.      |
| `sandbox.go`  | Reference `parseCCode` snippet. Delete once Parse works.                |

## How a language analyzer uses this package

```go
package java

import (
    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/java"

    "github.com/doITmagic/rag-code-mcp/internal/codetypes"
    "github.com/doITmagic/rag-code-mcp/internal/ragcode/analyzers/treesitterutil"
)

type Analyzer struct{ *treesitterutil.CodeAnalyzer }

func NewAnalyzer() *Analyzer {
    return &Analyzer{
        CodeAnalyzer: treesitterutil.NewCodeAnalyzer(treesitterutil.Options{
            Language:   java.GetLanguage(),
            Extensions: []string{".java"},
            Extract:    extractChunks,
        }),
    }
}

func extractChunks(root *sitter.Node, source []byte, path string) ([]codetypes.CodeChunk, error) {
    // walk the tree, emit one chunk per class/method/...
}
```

The embedded `*CodeAnalyzer` already satisfies `codetypes.PathAnalyzer`, so
`java.NewAnalyzer()` can be returned anywhere a `PathAnalyzer` is expected.

## Go patterns worth noting

The code uses a handful of idiomatic Go patterns. Quick tour:

### `sync.Pool` for expensive, non-concurrent objects

Tree-sitter parsers can't be shared across goroutines but are expensive to
build. `sync.Pool` lets us reuse them without writing lock code:

```go
pp.pool.New = func() any {
    p := sitter.NewParser()
    p.SetLanguage(lang)
    return p
}
```

`Get()` returns a warm parser (creating one if none cached); `Put()` makes
it available again. The runtime GC may evict pool entries under memory
pressure — that's fine, `New` will rebuild them on demand.

### Function values as configuration

```go
type Extractor func(root *sitter.Node, source []byte, path string) (...)

type Options struct {
    Extract Extractor
    ...
}
```

Storing a function as a struct field is the Go equivalent of a strategy
object. No interface, no subclass — each language hands in its own closure.

### Embedding for "inheritance-lite"

```go
type Analyzer struct{ *treesitterutil.CodeAnalyzer }
```

Methods on `*CodeAnalyzer` (`AnalyzePaths`, `AnalyzeFile`, ...) get promoted
to the outer struct. The Java analyzer doesn't override them — it just
inherits the behaviour and supplies an `Extractor`. If a language *does*
need custom path handling, it can shadow the method.

### Pointer receivers when you mutate

```go
func FillLocation(c *codetypes.CodeChunk, ...) { c.FilePath = ... }
```

Mutating fields requires `*CodeChunk`, not `CodeChunk` — value receivers
get a copy. Rule of thumb: use pointers for "modify me" arguments; values
for "read me" / small immutable types.

### Returning `(T, error)`

Every function that can fail returns `(value, error)` rather than throwing.
Idiomatic call site:

```go
tree, err := pool.Parse(ctx, source)
if err != nil {
    return nil, fmt.Errorf("parse %s: %w", path, err)
}
defer tree.Close()
```

`%w` wraps the inner error so callers can `errors.Is` / `errors.As` against
it. Plain `%v` would lose that information.

### Variadic parameters for "optional list"

```go
LeadingDocComment(n, source, "line_comment", "block_comment")
```

`commentTypes ...string` makes the trailing arguments optional and
type-checked. Inside the function it's a regular `[]string`.

### `defer` for cleanup

Always pair a `Get` with `defer Put` and an open `*Tree` with `defer
tree.Close()`. `defer` runs on function return regardless of how you exit
(early return, panic). This is the Go equivalent of RAII / `try-finally`.

## Implementation order (suggested)

1. **`ParserPool` (parser.go)** — start here; everything else depends on it.
   Implement `NewParserPool`, `Get`, `Put`, then `Parse`.
2. **`WalkSourceFiles` (walk.go)** — pure stdlib (`filepath.WalkDir`,
   `os.ReadFile`). No tree-sitter knowledge needed.
3. **`CodeAnalyzer` methods (analyzer.go)** — wire `WalkSourceFiles` and
   `ParserPool.Parse` together; call `opts.Extract`.
4. **Node helpers (node.go)** — thin wrappers; easy once you've seen the
   `*sitter.Node` API surface.
5. **Comment helpers (comment.go)** — pure string-handling.
6. **Chunk helpers (chunk.go)** — combine 1 + 4.
7. Delete `sandbox.go`.

After each step, run:

```bash
CGO_ENABLED=1 go build ./internal/ragcode/analyzers/treesitterutil/...
CGO_ENABLED=1 go test  ./internal/ragcode/analyzers/treesitterutil/...
```
