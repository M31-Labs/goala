# goala

A statically-typed, functional-leaning language that transpiles to standard Go.
Sealed types with **compile-time exhaustive pattern matching**, `Result`/`Option`,
`?` error propagation, expression-oriented syntax, and frictionless Go interop —
a real language on the M31 Labs stack, built to prove a point.

**The point:** goala's entire grammar is one Go function, authored with
[gotreesitter](https://github.com/odvcencio/gotreesitter)'s `grammargen` —
pure Go, no ANTLR, no JVM, no Bazel, no Node, no C toolchain. From that single
definition goala gets:

- a pure-Go parser (its own compiler front-end),
- a standard tree-sitter `parser.c` (ABI 14/15) — native support in
  **Neovim, Helix, Zed**, and every other tree-sitter editor,
- syntax highlighting queries,
- code intelligence (index, references, call graphs, MCP for AI agents) via
  [canopy](https://github.com/odvcencio/canopy),
- an LSP (`canopyls`) — all **inherited, not built**.

The transpile-to-Go backend is the cheap part. That's the thesis.

> **Status: design phase.** See [DESIGN.md](DESIGN.md) for the full
> architecture spec and phased build plan. Nothing below is implemented yet.

## Quick taste (planned)

```goala
package main

import (
    "fmt"
    "os"
    "strconv"
)

sealed Shape {
    Circle(radius float64),
    Rect(width float64, height float64),
}

func area(s Shape) float64 = match s {
    Circle(r)  => 3.14159 * r * r,
    Rect(w, h) => w * h,
}

func readPort(path string) Result[int] {
    let data = os.ReadFile(path)?            // Go (T, error) auto-lifted
    let port = strconv.Atoi(string(data))?
    Ok(port)
}

func main() {
    let shapes = []Shape{Circle(1.0), Rect(2.0, 3.0)}
    for s in shapes {
        fmt.Println(f"area: ${area(s)}")
    }
    match readPort("port.txt") {
        Ok(p)  => fmt.Println(f"port ${p}"),
        Err(e) => fmt.Println(f"failed: ${e}"),
    }
}
```

Delete a `match` arm and `goala build` fails with
`match on Shape is not exhaustive: missing Rect` — at compile time.

## Planned CLI

```bash
goala emit  file.goala          # transpile to Go on stdout
goala run   file.goala          # transpile + execute
goala build ./... -o bin/app    # compile a native binary
goala fmt   -w file.goala       # format
goala lint  file.goala          # lint
goala lsp                       # thin language server (canopyls covers cross-file)
goala grammar emit -c parser.c  # the thesis: emit a tree-sitter grammar
```

## Repository layout (planned)

```
grammar.go                # THE grammar — single source of truth
transpile*.go typeenv*.go # CST-walk transpiler + local inference
cmd/goala/                # CLI + LSP
register/                 # gotreesitter grammars registry hook (canopy inheritance)
dist/tree-sitter-goala/   # generated parser.c + queries (never hand-edited)
corpus/                   # sample programs + golden trees + golden Go
playground/               # wasm transpiler + CST viewer
demo/                     # the one-command thesis demo
```

## Lineage

- [ferrous-wheel](https://github.com/odvcencio/ferrous-wheel) — the template:
  a production-complete language front-end on grammargen (extends Go's grammar).
  goala authors a *distinct* grammar — the stronger proof.
  (sealed types, monads, do-notation) and the toolchain counter-example
  (ANTLR + JVM + Bazel).
- [gotreesitter](https://github.com/odvcencio/gotreesitter) /
  [canopy](https://github.com/odvcencio/canopy) — the stack goala inherits.

## License

MIT
