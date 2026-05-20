# treesitterutil

Shared scaffolding for tree-sitter–backed language analyzers (Java, Kotlin,
C, C++, ...). It is **not** a `PathAnalyzer` itself — it gives each language
analyzer a parser pool, a file walker, node helpers, comment-cleaning
helpers, and CodeChunk builders so they can stay small and focused on
grammar-specific extraction.

This package is **Phase 0** in `../../../../../Plan.md`.

## Status

Feature-complete for Phase 0. Every public function is implemented;
`go build` and `go vet` pass. `sandbox.go` still ships a one-call
`parseCCode` sample that predates `ParserPool.Parse`; delete it once the
first real C analyzer lands.

The package has no test file yet — add one when you write the first
language analyzer, since the pure-Go helpers (`CleanBlockComment`,
`WalkSourceFiles`) are easy to test without a grammar and the
grammar-touching helpers can be covered with a single bundled C fixture.

## Files

| File          | Contents                                                                            |
| ------------- | ----------------------------------------------------------------------------------- |
| `analyzer.go` | Package doc, `Options`, `Extractor`, `CodeAnalyzer`, `DefaultSkipDirs`.             |
| `parser.go`   | `ParserPool` (`sync.Pool` of `*sitter.Parser`) + one-shot `Parse`.                  |
| `walk.go`     | `WalkSourceFiles` — filesystem walker with skip-dirs / skip-tests / extension set.  |
| `node.go`     | `NodeText`, `StartLine`/`EndLine`, `FindChild(ren)ByType`, `WalkNamed`, `FindFirstNamed`. |
| `comment.go`  | `CleanLineComment`, `CleanBlockComment`, `LeadingDocComment`.                       |
| `chunk.go`    | `FillLocation`, `FillSelection`, `ExtractSignatureLine`.                            |
| `sandbox.go`  | Reference `parseCCode` snippet (pre-pool). Delete when a real C analyzer exists.    |

## How a language analyzer uses this package

```go
package c

import (
    sitter "github.com/smacker/go-tree-sitter"
    tsc    "github.com/smacker/go-tree-sitter/c"

    "github.com/doITmagic/rag-code-mcp/internal/codetypes"
    "github.com/doITmagic/rag-code-mcp/internal/ragcode/analyzers/treesitterutil"
)

type Analyzer struct{ *treesitterutil.CodeAnalyzer }

func NewAnalyzer() *Analyzer {
    return &Analyzer{
        CodeAnalyzer: treesitterutil.NewCodeAnalyzer(treesitterutil.Options{
            Language:   tsc.GetLanguage(),
            Extensions: []string{".c", ".h"},
            Extract:    extractChunks,
        }),
    }
}

func extractChunks(root *sitter.Node, source []byte, path string) ([]codetypes.CodeChunk, error) {
    var chunks []codetypes.CodeChunk
    treesitterutil.WalkNamed(root, func(n *sitter.Node) bool {
        switch n.Type() {
        case "function_definition":
            nameNode := treesitterutil.FindFirstNamed(n, func(c *sitter.Node) bool {
                return c.Type() == "identifier"
            })
            chunk := codetypes.CodeChunk{
                Type:      "function",
                Name:      treesitterutil.NodeText(nameNode, source),
                Language:  "c",
                Signature: treesitterutil.ExtractSignatureLine(n, source),
                Docstring: treesitterutil.LeadingDocComment(n, source, "comment"),
            }
            treesitterutil.FillLocation(&chunk, n, source, path)
            treesitterutil.FillSelection(&chunk, nameNode)
            chunks = append(chunks, chunk)
            return false // don't descend into the function body
        }
        return true
    })
    return chunks, nil
}
```

The embedded `*CodeAnalyzer` already satisfies `codetypes.PathAnalyzer`, so
`c.NewAnalyzer()` can be returned anywhere a `PathAnalyzer` is expected.

## Go patterns worth noting

The code uses a handful of idiomatic Go patterns. Quick tour:

### `sync.Pool` for expensive, non-concurrent objects

Tree-sitter parsers can't be shared across goroutines but are expensive to
build (they allocate C memory through CGo and bind a language). `sync.Pool`
lets us reuse them without writing lock code:

```go
pp.pool.New = func() any {
    p := sitter.NewParser()
    p.SetLanguage(lang) // closure captures lang
    return p
}
```

`Get()` returns a warm parser (creating one if none cached); `Put()` makes
it available again. The runtime GC may evict pool entries under memory
pressure — that's fine, `New` rebuilds them on demand.

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
to the outer struct. The C analyzer doesn't override them — it just
inherits the behaviour and supplies an `Extractor`. If a language *does*
need custom path handling, it can shadow the method.

### Pointer receivers when you mutate

```go
func FillLocation(c *codetypes.CodeChunk, ...) { c.FilePath = ... }
```

Mutating fields requires `*CodeChunk`, not `CodeChunk` — value receivers
get a copy. Rule of thumb: use pointers for "modify me" arguments; values
for "read me" / small immutable types.

### Returning `(T, error)` and wrapping with `%w`

Every function that can fail returns `(value, error)` rather than throwing.
Idiomatic call site:

```go
tree, err := pool.Parse(ctx, source)
if err != nil {
    return nil, fmt.Errorf("parse %s: %w", path, err)
}
defer tree.Close()
```

Use `%w` (and only `%w`) inside `fmt.Errorf` to *wrap* an error — the
chain is preserved so callers can `errors.Is(err, sitter.ErrOperationLimit)`
or `errors.As(err, &myErr)`. Use `%v` for plain *logging* via
`fmt.Fprintf(os.Stderr, ...)` — the chain stops there, only the rendered
string survives. Mixing the two compiles, but `go vet` flags `%w` outside
`fmt.Errorf`.

### Variadic parameters for "optional list"

```go
LeadingDocComment(n, source, "line_comment", "block_comment")
```

`commentTypes ...string` makes the trailing arguments optional and
type-checked. Inside the function it's a regular `[]string`.

### `defer` for cleanup

Always pair a `Get` with `defer Put` and an open `*Tree` with
`defer tree.Close()`. `defer` runs on function return regardless of how
you exit — normal return, error return, or panic. This is the Go
equivalent of RAII / `try-finally`.

### Closure capture for accumulator patterns

`WalkSourceFiles` and `WalkNamed` take a callback. The callback closes
over a slice declared in the caller's scope and appends to it — no need
to thread accumulators through return values:

```go
var out []codetypes.CodeChunk
WalkNamed(root, func(n *sitter.Node) bool {
    if n.Type() == "function_definition" {
        out = append(out, build(n))
        return false
    }
    return true
})
```

## Build / test

```bash
CGO_ENABLED=1 go build ./internal/ragcode/analyzers/treesitterutil/...
CGO_ENABLED=1 go vet   ./internal/ragcode/analyzers/treesitterutil/...
CGO_ENABLED=1 go test  ./internal/ragcode/analyzers/treesitterutil/...
```

CGo is mandatory: every tree-sitter grammar is a C library compiled via
CGo bindings, so a working `cc` is required on every builder. There is
no pure-Go fallback in this package.

## What's next

1. Write the first real language analyzer — `internal/ragcode/analyzers/c/` is
   the easiest, since `sandbox.go` already proves the C grammar wires up.
2. Register `LanguageC` in `internal/ragcode/language_manager.go`.
3. Extend the file-extension switch in `internal/workspace/manager.go` to
   pick up `.c` / `.h`.
4. Delete `sandbox.go`.
5. Repeat for Kotlin / Java / C++ in any order.
