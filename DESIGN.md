# goala — Design and Build Specification

**Status:** Design (pre-implementation)
**Version:** 0.1 draft, 2026-07-19
**Module path:** `m31labs.dev/goala`
**File extension:** `.goala`

---

## 0. Executive summary

goala is a statically-typed, functional-leaning language that transpiles to standard
Go. It is inspired by [gala](https://github.com/martianoff/gala) ("Scala on Go"):
sealed types with **compile-time exhaustive pattern matching**, `Option`/`Result`,
`?` error propagation, do-notation-style `bind`, expression-oriented syntax, and
frictionless Go interop.

But goala's *purpose* is not to be a better gala. goala exists to prove the
**grammargen thesis**:

> Define a new language's grammar ONCE — in pure Go, using
> `gotreesitter/grammargen` — and the entire front-end AND tooling ecosystem
> falls out for free. No ANTLR. No JVM. No Bazel. No Node. No C toolchain in the
> authoring loop.

Concretely, one `grammar.go` file yields:

| Artifact | Produced by | Consumed by |
|---|---|---|
| Pure-Go `*gotreesitter.Language` | `grammargen.GenerateLanguage` (encode.go:29) | goala's own parser, transpiler, CLI, LSP — in-process, no CGO |
| Serialized grammar blob (`.bin`) | `GenerateLanguageAndBlob` (encode.go:35) | embedding, the gotreesitter grammars registry |
| **`parser.c` (tree-sitter ABI 14/15)** | `GenerateC` / `EmitC` (codegen_c.go:13/:22) | Neovim, Helix, Zed — every tree-sitter editor, zero per-editor work |
| Go DSL source + resolved `grammar.json` | `EmitGrammarGo` / `ExportGrammarJSON` | interchange, review, upstream tree-sitter ecosystem |
| Inferred highlight query | grammargen highlight inference (`highlight_gen.go`) | editor syntax highlighting baseline |

And because `canopy` detects languages purely through the gotreesitter grammars
registry (`grammars.DetectLanguage`, gotreesitter/grammars/registry.go:249), the
moment goala ships a `register/` package (the ferrous-wheel trick,
ferrous-wheel/register/register.go), goala inherits:

- **Code intelligence**: `canopy index / search refs / graph calls / graph impact / mcp` across `.goala` files — the same engine already serving 206+ languages.
- **An LSP**: `canopyls` (canopy/cmd/canopyls) serving cross-file navigation off that index.
- **Highlighting + queries**: gotreesitter's query engine and highlight inference.

gala structurally cannot do this. Its ANTLR grammar produces exactly one
artifact — a JVM/Go parser for gala's own compiler — and every editor
integration (GoLand plugin with local ANTLR parsing, LSP configs for VS
Code/Neovim) is bespoke, hand-maintained work sitting on a Bazel + JVM
toolchain. goala's entire toolchain is `go build`.

The language backend — transpile-to-Go via CST walk — is deliberately the cheap
part. ferrous-wheel (`~/work/ferrous-wheel`) already proved that pattern to
production completeness: transpiler, type inference layer, formatter, 19-rule
linter, LSP, VS Code extension, browser playground, benchmark parity suite.
goala follows that template, sized down to a coherent gala-inspired subset and
up-leveled in one place ferrous-wheel is weak: **exhaustiveness is checked at
compile time, not runtime**.

---

## 1. The thesis, stated precisely

### 1.1 What gala's stack requires

| Layer | gala | goala |
|---|---|---|
| Grammar authoring | ANTLR4 `.g4` | Go DSL (`grammargen`) in `grammar.go` |
| Grammar build | ANTLR codegen on the JVM | `go build` (tables built at runtime or embedded blob) |
| Build system | Bazel (+ shipped Bazel rules) | `go build` / `go install` |
| Parser runtime | ANTLR runtime | gotreesitter (pure Go, no CGO) |
| Editor: JetBrains | Hand-built plugin embedding ANTLR parsing | not needed for tree-sitter editors; (VS Code-class editors get the LSP) |
| Editor: Neovim/Helix/Zed | LSP config only (no native grammar) | **emitted `parser.c` + `highlights.scm`** — native tree-sitter grammar |
| LSP | Hand-built `gala lsp` | Two tiers: inherited `canopyls` + a thin bespoke `goala lsp` |
| Code intelligence (refs, call graph, impact, MCP for agents) | none | **inherited from canopy** for free |
| Language count sharing this infra | 1 (gala) | 206+ (goala joins the registry) |

### 1.2 The three proof obligations

Everything in this spec serves one of these:

- **P1 — One grammar, every editor.** The same `Grammar()` value emits both the
  pure-Go parser goala's compiler uses and a standard `parser.c` (ABI 14/15)
  that Neovim/Helix/Zed load natively. No divergence possible: both are
  projections of one IR.
- **P2 — Inherited tooling.** Registering the grammar
  (`grammars.RegisterExtension`) is sufficient for canopy indexing, canopy code
  intel commands, canopy's MCP server (AI agents can navigate goala code), and
  the canopyls LSP to work on `.goala` files. Zero goala-specific code inside
  canopy.
- **P3 — The backend is the cheap part.** The transpiler is a CST walk emitting
  standard Go, with the Go compiler as the semantic backstop. The goala-specific
  checker is a thin obligations layer (exhaustiveness, `?` contexts, sealed
  constructor arity). Prior art: ferrous-wheel's `transpile.go` + `typeenv*.go`.

### 1.3 Why this matters (the pitch behind the pitch)

Language front-ends are historically the expensive, ecosystem-fragmenting part
of making a language: parser + one grammar file per editor + an LSP + an
indexer, each drifting from the others. grammargen collapses all of it to one
Go function. goala is the end-to-end demonstration on a *new* language — not an
extension of Go's grammar (ferrous-wheel already proved that) but a language
authored from `NewGrammar("goala")` up.

---

## 2. Language surface

### 2.1 Design stance

- **Coherent subset of gala's ideas, not a clone.** Everything included must be
  (a) buildable by a CST-walking transpiler with local type inference, and
  (b) load-bearing for the thesis demo.
- **Expression-oriented.** `match`, `if`, and blocks are expressions; function
  bodies may be `= expr`; the last expression of a block is its value.
- **Go-shaped where it touches Go.** Type expressions, literals, imports,
  packages, and visibility (exported-iff-capitalized) are Go's, verbatim. Any
  Go type is nameable in goala; any Go package is importable with no bindings.
- **Zero runtime.** Generated code is standard Go with no goala import. Helpers
  (`Result`, `Option`, constructors) are injected into the generated file only
  when used, with collision avoidance — the ferrous-wheel policy
  (ferrous-wheel README "Design notes").

### 2.2 Feature set (v1, normative)

#### Declarations

```goala
package geometry

import (
    "fmt"
    "math"
)

// Sealed sum type: the flagship feature.
// Cases are records; the set of cases is closed at the package boundary.
sealed Shape {
    Circle(radius float64),
    Rect(width float64, height float64),
    Point,                                  // nullary case
}

// Concise record (product type). Transpiles to a Go struct + positional constructor.
struct Vec(x float64, y float64)

// Functions: Go-style signatures, optional `= expr` single-expression bodies.
func area(s Shape) float64 = match s {
    Circle(r)  => math.Pi * r * r,
    Rect(w, h) => w * h,
    Point      => 0,
}

// Methods: Go receiver syntax, expression bodies allowed.
func (v Vec) norm() float64 = math.Sqrt(v.x*v.x + v.y*v.y)

// derive: auto-generated impls (v1: Stringer only).
derive Stringer for Shape

// Plain Go type declarations remain available for aliases and interop.
type Meters = float64
type Handler interface {
    Handle(msg string) Result[int]
}
```

#### Bindings and expressions

```goala
func demo() {
    let x = 42                       // immutable; reassignment is a compile error
    var count = 0                    // mutable
    count = count + 1

    let label = if count > 0 { "items" } else { "empty" }   // if-expression

    let double = (n int) => n * 2    // lambda; param types inferable from context
    let inc = n => n + 1

    let msg = f"count=${count}, doubled=${double(count)}"   // f-string interpolation
    fmt.Println(msg, label, inc(1))
}
```

#### Option, Result, `?`, and Go interop

```goala
package main

import (
    "fmt"
    "os"
    "strconv"
)

// Result[T] and Option[T] are built-in sealed-style types (zero-runtime, injected on use).
func readPort(path string) Result[int] {
    let data = os.ReadFile(path)?            // Go (T, error) call auto-lifted; ? propagates Err
    let port = strconv.Atoi(string(data))?   // chainable
    if port <= 0 { return Err[int](fmt.Errorf("bad port %d", port)) }
    Ok(port)                                 // last expression is the return value
}

func lookup(env map[string]string, key string) Option[string] {
    match env[key] {                          // Go map read; comma-ok lifted to Option
        Some(v) => Some(v),
        None    => None[string](),
    }
}

func main() {
    match readPort("port.txt") {
        Ok(p)  => fmt.Println(f"listening on ${p}"),
        Err(e) => fmt.Println(f"failed: ${e}"),
    }
}
```

Interop rules (normative):

1. Calling a Go function returning `(T, error)`: in `?` position it propagates
   (`Err` in a `Result` function, or `(zero, err)` return in a plain
   `(T, error)` goala function); as a `match` subject it is lifted to
   `Result[T]`; anywhere else, using it single-valued is a compile error with a
   fix-it ("add `?` or match on it").
2. Calling a Go function returning `(T, bool)` (comma-ok): as a `match` subject
   or in `?` position it lifts to `Option[T]`.
3. goala functions compile to ordinary Go functions in the same package —
   **hand-written Go in the same module can call goala code** and vice versa.
   `Result[T]`/`Option[T]` in an exported goala signature are ordinary generic
   Go structs in the output, usable from Go.

#### `bind` — do-notation (thin sugar)

```goala
func fetchReceipt(id int) Result[Receipt] {
    bind order   = fetchOrder(id)        // == let order = fetchOrder(id)?
    bind valid   = validate(order)
    bind payment = charge(valid)
    Ok(Receipt(order.Id, payment))
}
```

`bind x = e` is defined as exactly `let x = e?`. It exists to preserve the gala
lineage in demos and reads better in monadic pipelines. It is *not* generic
do-notation: v1 `bind` works only in functions returning `Result[T]`,
`Option[T]`, or `(T, error)`. (Scope discipline; see 2.4.)

#### Pattern matching (normative semantics)

- Arm forms: `Constructor(binders...)`, `Constructor`, literal, binder
  (`n => ...`), wildcard `_`, each with an optional guard `if expr`.
- Arm bodies: expression or block.
- **Exhaustiveness is a compile-time check** when the subject's type resolves to
  a sealed type, `Result`, `Option`, or a lifted interop call:
  - Every case must be covered by an *unguarded* arm, or an unguarded
    binder/wildcard arm must be present.
  - Guarded arms never count toward coverage.
  - Missing cases are reported by name:
    `match on Shape is not exhaustive: missing Rect, Point`.
- If the subject's type cannot be resolved (untyped interop edge), goala
  *requires* a wildcard arm and reports why — graceful degradation, never a
  silent runtime hole. (This is the deliberate upgrade over ferrous-wheel,
  whose `match` panics at runtime on unmatched values.)
- v1 patterns are one constructor level deep. Nested constructor patterns
  (`Ok(Circle(r))`) are deferred (see 2.4).

#### Control flow

```goala
for s in shapes { ... }          // range over slice/map/channel
for i, s in shapes { ... }
for count < 10 { ... }           // Go's condition-for
return, break, continue          // Go semantics
```

#### Generics

Go-style, passed through: `func head[T any](xs []T) Option[T]`,
`sealed Tree[T any] { Leaf(value T), Node(left Tree[T], right Tree[T]) }`.
Constraint syntax is Go's. No inference beyond what the transpiler needs to
emit valid Go (the Go compiler enforces the rest).

### 2.3 A complete sample program (the demo corpus anchor)

```goala
package main

import (
    "fmt"
    "os"
    "strings"
)

sealed Token {
    Word(text string),
    Number(value int),
    Punct(ch string),
}

struct Stats(words int, numbers int, puncts int)

func classify(raw string) Token = match raw {
    r if isNumeric(r) => Number(mustAtoi(r)),
    r if len(r) == 1 && strings.ContainsAny(r, ".,;:!?") => Punct(r),
    r => Word(r),
}

func tally(tokens []Token) Stats {
    var s = Stats(0, 0, 0)
    for t in tokens {
        match t {
            Word(_)   => { s.words = s.words + 1 },
            Number(_) => { s.numbers = s.numbers + 1 },
            Punct(_)  => { s.puncts = s.puncts + 1 },
        }
    }
    s
}

func load(path string) Result[[]Token] {
    let data = os.ReadFile(path)?
    let fields = strings.Fields(string(data))
    var tokens = []Token{}
    for f in fields {
        tokens = append(tokens, classify(f))
    }
    Ok(tokens)
}

func main() {
    match load("input.txt") {
        Ok(tokens) => {
            let s = tally(tokens)
            fmt.Println(f"words=${s.words} numbers=${s.numbers} puncts=${s.puncts}")
        },
        Err(e) => fmt.Println(f"error: ${e}"),
    }
}
```

This exercises: sealed types, guards, exhaustive match, records, `?` on Go
interop, f-strings, expression bodies, last-expression returns, Go stdlib
usage. It is the first file in the parse corpus (Phase 1) and the first program
the transpiler must round-trip (Phase 2).

### 2.4 Deliberately NOT included (scope discipline)

| Excluded | Why | gala has it? |
|---|---|---|
| `Future`, `IO`, `Either`, `Validated`, `Try` as distinct types | `Result`/`Option` cover the demo; effect stacks need HKT-ish machinery a CST transpiler shouldn't fake. `Try[T]` ≅ `Result[T]` here. | yes |
| Generic do-notation / structural `FlatMap` resolution / `also` parallel binds | Requires real typeclass dispatch; v1 `bind` is monomorphic sugar. | yes |
| Zero-reflection JSON (`Codec[T]`/`StructMeta`) | Great feature, orthogonal to the thesis. `derive Json` is a listed post-v1 extension — the ferrous-wheel `derive` machinery shows the path. | yes |
| Immutable collections library (`List`, `HashMap`, `TreeSet`…) | Library work, not thesis work. Go slices/maps + `for`/`match` suffice. | yes |
| Named arguments with defaults | Needs call-site rewriting with full signature knowledge everywhere; cut. | yes |
| Nested constructor patterns, regex extractors, `Tuple2..5` | Exhaustiveness for nested patterns is a product-space construction; post-v1. Multi-return covers tuples. | yes |
| Concurrency sugar (`fan out`, `select!`, actors) | ferrous-wheel owns that space; goala stays functional-core. | partly |
| Macros, compiler plugins, reflection | Never. | no |
| Its own module system | Reuses `go.mod` outright (see 4.5). gala reimplements `mod init/add/tidy`; goala inherits Go's. | yes (`gala mod`) |

---

## 3. Grammar plan

### 3.1 Decision: a DISTINCT grammar (not an extension of Go's)

**Decision: `NewGrammar("goala")` — author goala as a standalone grammar that
*borrows* Go's lexical and type sub-grammars, rather than
`ExtendGrammar(..., GoGrammar(), ...)`.**

Reasoning:

1. **Thesis strength.** ferrous-wheel already proved `ExtendGrammar` to
   production quality (~80 rules over Go's 116, ferrous-wheel/grammar.go:32).
   The unproven claim is that grammargen is a *language-authoring* tool for new
   languages — first defined rule is the start rule, conflicts diagnosed by
   `GenerateWithReport`, `parser.c` emitted for a grammar that has never
   existed in the tree-sitter ecosystem. A Go-superset grammar would prove
   nothing new.
2. **Surface fit.** goala's expression-oriented forms (`func f(x int) int = expr`,
   `match` as an expression with `=>` arms, last-expression blocks, `sealed`
   declarations, arrow lambdas) *fight* Go's statement grammar. Extending Go
   means inheriting ~116 rules of statement-oriented shape and then overriding
   the spine — more conflict surface than authoring the spine natively
   (ferrous-wheel's contortions around Go's float literal token for `0..10`,
   grammar.go:824-886, show the cost of fighting a base grammar's lexer).
3. **Borrowing keeps the interop guarantee cheap.** goala's *types, literals,
   imports, comments* are Go's by design (2.1). Those sub-grammars are lifted
   from `GoGrammar()` as cloned rule subtrees, so "any Go type is nameable"
   holds by construction and we don't re-author Go's hairy literal tokens.

Mechanics: a small `ImportRules(dst, src *Grammar, roots ...string)` helper
clones each root rule and its transitive `Sym()` dependencies from
`GoGrammar()` into the goala grammar — the same deep-copy discipline
`ExtendGrammar` itself uses (grammargen/grammar.go:322-379, `cloneRule` loop).
This helper is generic and should be upstreamed into grammargen next to
`ExtendGrammar` (it is the "compose grammars by subtree" primitive the thesis
deserves).

Borrowed roots (initial list): `identifier`, `comment`, `int_literal`,
`float_literal`, `rune_literal`, `interpreted_string_literal`,
`raw_string_literal`, `_type` (and its closure: pointer/slice/array/map/
channel/function/qualified/generic-instantiation types), `import_declaration`
closure, `package_clause`.

Authored fresh (the goala spine), roughly 60–80 rules:

| Group | Rules (representative) |
|---|---|
| Structure | `source_file`, `_declaration`, `sealed_declaration`, `sealed_case`, `struct_declaration`, `function_declaration`, `method_receiver`, `derive_declaration`, `go_type_declaration` |
| Statements | `_statement`, `let_declaration`, `var_declaration`, `bind_declaration`, `assignment`, `return_statement`, `for_in_statement`, `for_statement`, `expression_statement`, `block` |
| Expressions | `_expression`, `match_expression`, `match_arm`, `if_expression`, `lambda_expression`, `call_expression`, `selector_expression`, `index_expression`, `composite_literal`, `binary_expression` (Go's precedence table via `PrecLeft` levels), `unary_expression`, `try_expression` (postfix `?`), `interpolated_string` (`f"..."` with `${}` parts) |
| Patterns | `_pattern`, `constructor_pattern`, `literal_pattern`, `binder_pattern`, `wildcard_pattern`, `guard` |

Grammar-level settings:

- `g.SetWord("identifier")` — keyword extraction (keywords: `package import func
  sealed struct type interface let var bind match if else for in return break
  continue derive`).
- `g.SetSupertypes("_expression", "_statement", "_declaration", "_pattern")` —
  richer node metadata for queries and canopy.
- `g.SetExtras(Sym("comment"), Pat("[ \\t\\r\\n]"))` — copy ferrous-wheel's
  hard-won lesson verbatim (grammar.go:800-818): restrict extras to Go's real
  whitespace so verbatim source echoes never smuggle `\v`/`\f` into emitted Go.
- Embedded `g.Test(...)` cases per rule cluster (grammargen runs them in
  `doctor`).

### 3.2 Known ambiguities and the conflict workflow

Anticipated conflicts (declare intentionally, resolve the rest):

1. **Arrow lambda vs parenthesized expression**: `(x) => x + 1` vs `(x)`.
   Classic; grammargen already ships a TypeScript grammar
   (grammargen/typescript_grammar.go) where this exact ambiguity is handled via
   declared conflicts + dynamic precedence. Copy that resolution.
2. **Constructor pattern vs call expression** inside `match` arms:
   `Circle(r) => ...` — `Circle(r)` parses like a call. Resolve by making
   `_pattern` its own sub-grammar reachable only from `match_arm` (patterns are
   not expressions), with a declared conflict where a bare binder overlaps an
   identifier expression in guard position.
3. **`match` as expression vs statement position** and block-vs-composite-
   literal after `if cond` — resolve like Go does (no composite literal in
   condition position) via a `_simple_expression` restriction, or a declared
   conflict + dynamic precedence.
4. **Generic instantiation vs index**: `f[T](x)` vs `xs[i]` — inherited with
   Go's `_type`/expression rules and Go's own declared conflicts; borrowing
   Go's rules imports Go's resolution.

Workflow (the thesis showcase for grammar diagnostics):

- Every grammar change runs `GenerateWithReport`
  (grammargen/diagnostics.go:1085) in a unit test; the test asserts
  `report.Conflicts` contains only the *whitelisted, commented* conflict set.
  New unexplained `ConflictDiag`s fail CI.
- Interactive loop: `go run ./cmd/grammargen doctor -grammar` equivalents via a
  `goala grammar doctor` subcommand wrapping the same APIs, plus
  `goala grammar parse -text '...' -format sexpr` for tree feedback and
  `-write-expect/-expect` golden snapshots (mirroring grammargen's README
  commands).

### 3.3 The four emit targets (thesis centerpiece: `parser.c`)

`goala grammar emit` produces, from the single `Grammar()` value:

```
goala grammar emit -c        dist/tree-sitter-goala/src/parser.c   # ABI 14/15 C parser
goala grammar emit -bin      dist/goala.bin                        # gotreesitter blob
goala grammar emit -json     dist/grammar.json                     # resolved tree-sitter grammar.json
goala grammar emit -go       grammar_generated.go                  # Go DSL round-trip (review artifact)
goala grammar emit -highlight queries/highlights.scm               # inferred highlight baseline
```

The `parser.c` path is `GenerateC(g)` → `GenerateLanguage` → `EmitC`
(codegen_c.go:13-48): symbol/field enums, parse tables, lex functions, ABI
export — the same emitter already parity-tested against upstream tree-sitter
across the top-50 grammars (grammargen's `*_parity_test.go` suite). The
`dist/tree-sitter-goala/` directory is laid out as a standard tree-sitter
grammar repo (`src/parser.c`, `queries/highlights.scm`, `queries/locals.scm`,
`queries/tags.scm`) so every editor's normal grammar-install path just works —
but it is 100% generated; **the repo's source of truth is `grammar.go`**.

`queries/` policy: start from the inferred highlight query, then hand-tune a
committed `highlights.scm` (captures for keywords, `sealed_case` names as
`@constructor`, f-string interpolation, `@function`, `@type`). The tuned file
ships in both the editor dist and the `register/` entry.

### 3.4 Registry integration (the P2 lever)

`goala/register/register.go`, exactly the ferrous-wheel shape
(ferrous-wheel/register/register.go:19-31):

```go
func init() {
    grammars.RegisterExtension(grammars.ExtensionEntry{
        Name:             "goala",
        Extensions:       []string{".goala"},
        Aliases:          []string{"goala"},
        GenerateLanguage: func() (*gotreesitter.Language, error) { return goala.Language() },
        HighlightQuery:   goala.HighlightQuery(),
    })
}
```

Blank-importing this package makes `grammars.DetectLanguage("x.goala")`
(registry.go:249, extension map hit at :273-277) return goala — which is the
single hook canopy uses everywhere (canopy/pkg/scope/index.go:17,
canopy/pkg/complexity/complexity.go:86, canopy/internal/scope/scope.go:63,
…). It lives in a subpackage so the goala core stays free of the ~22 MB
grammars registry — only consumers that want detection pay for it (the
ferrous-wheel rationale, register.go doc comment). Once stable, submit the
registration upstream to gotreesitter's registry (as `fw`/`danmuji` aliases
already are, registry.go:414) so stock canopy builds index goala with no
custom build at all.

---

## 4. Transpiler architecture

### 4.1 Pipeline

```
.goala source
  → gotreesitter parse (pure-Go Language, cached via sync.Once — fw transpile.go:23-28 pattern)
  → parse diagnostics (ERROR/MISSING nodes → user-facing errors with spans; fw parse_diagnostics.go)
  → lint (blocking errors / non-blocking warnings; fw lint.go — runs before transpile)
  → semantic pass (typeenv: collect → infer → resolve; sealed registry; obligations checks)
  → emit pass (CST walk, per-node emitters, string builder)
  → helper injection (Result/Option/derive impls, only-if-used, collision-checked)
  → go/format.Source
  → sourcemap (goala line/col ↔ generated Go line/col; fw sourcemap.go)
  → [run|build] write to temp module or sibling files, invoke go toolchain,
    map Go compiler errors back through the sourcemap
```

Key architectural stance (P3): **the Go compiler is the real type checker.**
goala's semantic pass only discharges goala-specific obligations:

1. Exhaustiveness of `match` over sealed/`Result`/`Option`/lifted subjects.
2. `?` / `bind` context validity (enclosing function returns `Result`,
   `Option`, or `(T, error)`; correct lift for the operand's shape).
3. Sealed constructor arity/field checks and "sealed cases only in the
   declaring package".
4. `let` immutability enforcement.
5. Interop lifts ((T,error) / (T,bool) call classification).

Everything else — assignability, generics, method sets, unused imports — is Go
compiler territory, surfaced through the sourcemap so errors point at `.goala`
lines. This keeps the checker a few thousand lines, not a compiler.

### 4.2 Type/semantic layer

Modeled on ferrous-wheel's `typeenv` family (typeenv.go, typeenv_collect.go,
typeenv_infer.go, typeenv_unify*.go, typeenv_resolve.go, typeenv_imports.go):

- **Collect**: walk declarations; record function signatures, sealed types with
  their case lists, struct records, let/var bindings with declared or inferred
  types.
- **Infer**: local, bidirectional inference with a small unifier — enough to
  resolve `match` subject types, lambda parameter types from call-site context,
  and `?` operand shapes. No global Hindley–Milner.
- **Imports**: load imported Go package signatures via `go/packages` +
  `go/types` (fw's typeenv_imports.go precedent) so interop lifts and
  `(T, error)` detection are signature-accurate. Cache aggressively
  (module-level cache keyed by package path + go.sum). Degradation mode when
  load fails: `?` on an unknown call assumes `(T, error)`; `match` on unknown
  requires a wildcard arm.
- **Sealed registry + exhaustiveness**: per package, `map[sealedName][]caseName`;
  per match, coverage set from unguarded arms; diagnostic lists missing cases.

### 4.3 Lowerings (normative Go output)

**Sealed type** (`sealed Shape { Circle(radius float64), Rect(width, height float64), Point }`):

```go
type Shape interface{ isShape() }

type Circle struct{ Radius_ float64 }        // NO — see naming rule below
```

Naming rule: goala field and case names map to Go **verbatim** (no case
munging); visibility is therefore Go's capitalization rule, taught in the docs.
The actual emission:

```go
type Shape interface{ isShape() }

type Circle struct{ radius float64 }
func (Circle) isShape() {}

type Rect struct{ width, height float64 }
func (Rect) isShape() {}

type Point struct{}
func (Point) isShape() {}
```

Constructor *calls* `Circle(2.0)` emit composite literals `Circle{radius: 2.0}`
directly (no constructor funcs needed in the common case; a `func NewCircle`
is emitted only when the constructor is used as a value, e.g. passed to `Map`).

**Match over sealed** (expression position → IIFE, statement position → plain
switch; fw's if-expression IIFE precedent):

```go
// match s { Circle(r) => A, Rect(w,h) => B, Point => C }
func() float64 {
    switch __m := s.(type) {
    case Circle:
        r := __m.radius
        return A
    case Rect:
        w, h := __m.width, __m.height
        return B
    case Point:
        return C
    }
    panic("unreachable: exhaustiveness checked at compile time")
}()
```

The trailing panic is provably dead (checker guarantees coverage) but keeps the
Go compiler satisfied about return paths.

**Result/Option representation** (injected only when used; `__goala`-prefixed
internals; collision detection against user declarations of
`Result/Option/Ok/Err/Some/None`, per fw README):

```go
type Result[T any] struct { ok bool; val T; err error }
func Ok[T any](v T) Result[T]      { return Result[T]{ok: true, val: v} }
func Err[T any](e error) Result[T] { return Result[T]{err: e} }

type Option[T any] struct { ok bool; val T }
func Some[T any](v T) Option[T] { return Option[T]{ok: true, val: v} }
func None[T any]() Option[T]    { return Option[T]{} }
```

Struct-based (no interface, no allocation), so `match` on Result/Option lowers
to `if r.ok { ... } else { ... }` rather than a type switch, and exported goala
APIs remain pleasant to call from Go.

**`?` / `bind`** in a `Result[int]` function, operand a Go `(T, error)` call:

```go
// let data = os.ReadFile(path)?
__v0, __err0 := os.ReadFile(path)
if __err0 != nil {
    return Err[int](__err0)
}
data := __v0
```

Operand already `Result[U]`: unwrap `.ok/.val/.err`. Enclosing function
returning `(T, error)`: propagate `return *new(T), __err0` style zero values.
Enclosing `Option`: `(T, bool)` and `Option` operands only (`error` operands in
an Option context are a compile error with a fix-it).

**Records**: `struct Vec(x float64, y float64)` → `type Vec struct { x, y float64 }`;
`Vec(1, 2)` → `Vec{x: 1, y: 2}`.

**f-strings**: `f"port ${p}"` → `fmt.Sprintf("port %v", p)` with `fmt`
auto-imported only when used (fw auto-import policy).

**Expression bodies / last-expression return**: `= expr` bodies and trailing
block expressions lower to `return expr`.

**derive Stringer for Shape**: emits `func (s Shape) String() string`? No —
Shape is an interface; emit a `String()` method per case struct returning
`"Circle(2.5)"`-style renderings. (fw's derive machinery is the template.)

### 4.4 Runtime-library policy

Zero-runtime, hard requirement (matching gala's "transpiles to plain Go" and
fw's "no runtime library, no hidden dependencies"):

- Generated files import only stdlib packages actually used.
- All helpers are emitted into the generated output, guarded by usage
  detection (fw's `transpile_support.go` / `transpile_support_detection_test.go`
  pattern).
- Multi-file packages emit helpers once into a `zz_goala_support.go` sibling.
- A generated-file header (`// Code generated by goala — DO NOT EDIT.`) on
  every output, and outputs always pass `gofmt -l` clean.

### 4.5 Build model and packages

- **Single-file mode** (Phase 2): `goala emit/run/build file.goala` — fw CLI
  semantics (fw cmd/ferrous-wheel/main.go), temp-module execution for `run`.
- **Package mode** (Phase 3): a directory of `.goala` files inside a normal Go
  module transpiles to sibling `<name>_goala.go` files in the same package.
  Consequences, all deliberate:
  - `go.mod` is the module system; `go get` is dependency management. goala
    ships no `goala mod` (contrast: gala reimplements `mod init/add/tidy`).
  - Hand-written `.go` and `.goala` files coexist in one package —
    bidirectional interop by construction.
  - `//go:generate goala emit ./...` makes goala a normal Go codegen citizen;
    CI runs `goala build ./... && git diff --exit-code` for drift.
- Cross-file, same-package sealed exhaustiveness works because the collect pass
  reads the whole package. Cross-package sealed extension is prohibited
  (checker error), which is what makes exhaustiveness sound.

---

## 5. The thesis demo (the GopherCon artifact)

One command, from a clean machine with only Go, a C compiler, Neovim, and
canopy installed:

```
make demo        # in the goala repo
```

Script (`demo/run.sh`), each step printed with narration:

1. **"This is the entire front-end."** `bat grammar.go` — one Go file, no
   `.g4`, no `grammar.js`, no Bazel, no JVM, no Node.
2. **Grammar → every artifact.** `goala grammar emit` writes `parser.c`
   (+ line count!), `grammar.json`, blob, `highlights.scm` — timed; the whole
   emit is a `go run`, seconds not minutes.
3. **Native editor grammar, no toolchain.**
   `cc -shared -fPIC -I demo/treesitter-headers dist/tree-sitter-goala/src/parser.c -o ~/.local/share/nvim/site/parser/goala.so`
   plus copying `queries/goala/highlights.scm`; then
   `nvim demo/tokens.goala` opens with full tree-sitter highlighting and
   `:InspectTree` shows the live CST. **No tree-sitter CLI, no Node, no
   nvim-treesitter compile step — just cc on an emitted file.** (Helix and Zed
   variants in `demo/helix/` and `demo/zed/` prove "every editor" — same
   `parser.c`, their normal grammar config.)
4. **Inherited code intelligence.** `canopy index build demo/project && canopy
   search refs classify demo/project && canopy graph calls load
   demo/project` — canopy indexes `.goala`, finds references, walks the call
   graph. Then `canopy mcp --root demo/project` and an agent asks "what calls
   classify?" — **AI-agent code intel for a language that didn't exist this
   morning.**
5. **Inherited LSP.** `canopyls` attached to the same project in Neovim:
   go-to-definition and references over `.goala` buffers.
6. **The backend is the cheap part.** `goala run demo/tokens.goala` — output
   prints; `goala emit demo/tokens.goala | bat -l go` shows clean, zero-runtime
   Go. Delete a match arm, `goala build` fails with
   `match on Token is not exhaustive: missing Punct` — the gala flagship
   feature, no JVM in sight.
7. **The punchline slide.** Side-by-side toolchain inventory: gala = ANTLR4 +
   JVM + Bazel + hand-built IDE plugin + hand-built LSP, editors get zero
   native grammar support. goala = `go build`, and Neovim/Helix/Zed/canopy/LSP
   all came from one function.

The single sharpest artifact is **step 3**: watching a brand-new language get
native syntax highlighting and `:InspectTree` in stock Neovim from an emitted
`parser.c`, ~30 seconds after showing the Go function that defines the
grammar. Everything else orbits that moment.

Demo assets to build (Phase 5): `demo/run.sh`, `demo/tokens.goala`,
`demo/project/` (3–4 file package with a call graph worth showing),
`demo/treesitter-headers/` (vendored `tree_sitter/parser.h`), Helix/Zed
configs, and an asciinema recording as the durable artifact.

---

## 6. Tooling surface

Matching ferrous-wheel's completeness bar, with the inherited/bespoke split
made explicit (the split IS the thesis):

| Tool | Status | Notes |
|---|---|---|
| `goala emit <file|./...>` | bespoke | Go source to stdout / sibling files; generated header; sourcemap sidecar with `-sourcemap` |
| `goala run <file>` | bespoke | transpile + `go run` in temp module; exit code passthrough (fw main.go semantics) |
| `goala build <file|./...> [-o out]` | bespoke | transpile + `go build`; Go errors mapped through sourcemap |
| `goala fmt [-w|--check]` | bespoke | CST-based formatter (fw format.go pattern); idempotence-gated |
| `goala lint` | bespoke | rule engine, errors block transpile (fw lint.go); launch set ~10 rules (shadowed binding, unused let, non-exhaustive-adjacent smells, guard-only match, unreachable arm, `?` outside Result fn, …) |
| `goala lsp` | bespoke (thin) | fw lsp.go scope: diagnostics on open/change (parse + lint + obligations), hover (inferred types), document symbols, definition within file/package. Single-file fast path; canopyls is the cross-file tier |
| `canopyls` | **inherited** | cross-file defs/refs off the canopy index; zero goala code |
| `canopy` CLI + MCP | **inherited** | index/search/graph/analyze/mcp via `register/` |
| Neovim/Helix/Zed | **inherited** | emitted `parser.c` + queries; per-editor cost is a config file |
| VS Code extension | bespoke (small) | fw editor/vscode template: TextMate baseline + LSP client + snippets; VS Code is the one editor that can't consume `parser.c`, and that fact goes in the demo narration |
| Playground | bespoke | fw playground pattern: wasm build exporting `goalaTranspile` (fw playground/wasm/main.go:19-44) + backend sandbox for run (fw playground/backend); **plus a CST pane** — parse-to-sexpr in wasm, showing the same grammar driving the browser. GitHub Pages deploy |
| `goala grammar emit|doctor|parse` | bespoke (thin wrappers) | over grammargen APIs; the demo's front door |

Repository layout (mirrors ferrous-wheel):

```
goala/
  grammar.go                 # THE grammar (source of truth)
  grammar_import.go          # ImportRules borrowing from GoGrammar()
  grammar_test.go            # GenerateWithReport conflict whitelist + embedded tests
  language.go                # cached Language(), HighlightQuery()
  transpile*.go              # emitters
  typeenv*.go                # collect/infer/resolve/imports
  sealed.go exhaustive.go    # obligations layer
  format.go lint.go          # fmt + lint
  sourcemap.go parse_diagnostics.go
  corpus/                    # .goala + golden .sexpr + golden .go + run outputs
  cmd/goala/                 # CLI (main.go, lsp.go, grammar.go)
  register/                  # grammars.RegisterExtension
  dist/tree-sitter-goala/    # generated: src/parser.c, queries/ (committed, CI-checked for drift)
  editor/vscode/
  playground/{wasm,backend,web}/
  bench/                     # goala-vs-handwritten-Go parity (fw bench/ pattern)
  demo/
```

---

## 7. Phased build plan

Sizing assumes one experienced engineer driving agent workers; calendar
estimates are conservative single-threaded figures.

### Phase 0 — Syntax freeze via corpus (3–4 days)

Deliverables:
- `corpus/` with **12 sample programs** written first, covering every 2.2
  feature (including the 2.3 anchor program, an interop-heavy sample using
  `net/http` + `strconv`, a generics sample, and 3 deliberately-invalid
  samples for future diagnostics).
- `SYNTAX.md`: one page freezing token-level decisions (lambda arrow, f-string
  delimiters, trailing commas, comment forms).
- Repo scaffold, `go.mod`, CI skeleton.

Gate: corpus reviewed and frozen; every later phase parses/compiles these
exact files. (Corpus-first is the cheap insurance against grammar churn.)

### Phase 1 — Grammar + pure-Go parser + parser.c (2–3 weeks)

Deliverables:
- `grammar.go` + `ImportRules` helper (proposed upstream to grammargen).
- `grammar_test.go`: `GenerateWithReport` conflict whitelist (every declared
  conflict has a comment explaining it), embedded `g.Test` cases green.
- Golden S-expression snapshots for all corpus files (`-write-expect` style).
- `goala grammar emit` subcommand producing all five artifacts (3.3).
- **C parity harness**: compile emitted `parser.c` against vendored tree-sitter
  runtime headers, parse the corpus through the C runtime, byte-compare
  S-expressions with the pure-Go parse (grammargen's `*_parity_test.go`
  discipline, applied to goala).
- Tuned `queries/highlights.scm`.
- `register/` package; smoke test: `grammars.DetectLanguage("x.goala")` resolves
  and parses.

Gate (all executable):
`go test ./...` green including: 0 unexplained conflicts; 12/12 corpus files
parse ERROR-free with golden trees; C-runtime parity 12/12; `cc` compiles
`parser.c` warning-clean; highlight query compiles under gotreesitter's query
engine. **Neovim smoke: manual checklist showing highlighting on the anchor
file.**

### Phase 2 — Transpiler MVP (2 weeks)

Deliverables:
- Emit pass for the full 2.2 surface *minus* checks: sealed lowering, match
  (assume exhaustive), records, let/var, `?` with syntactic (T,error)
  assumption, f-strings, lambdas, expression bodies, helper injection,
  `go/format`, generated headers.
- `goala emit/run/build` single-file mode; temp-module `run`.
- Golden Go output tests + end-to-end compile-and-run tests for 9 valid corpus
  programs (expected stdout captured).

Gate: 9/9 corpus programs → `go build`-clean Go → run with expected output;
golden `.go` files gofmt-stable; `go vet` clean on all generated output.

### Phase 3 — Semantic layer + package mode (3–4 weeks)

Deliverables:
- typeenv (collect/infer/resolve), `go/packages` import signature loading with
  cache and degradation mode.
- Obligations checks: exhaustiveness (with missing-case naming), `?`/`bind`
  context validity, sealed arity/package rules, immutability.
- Sourcemap + Go-compiler error mapping (`goala build` reports `.goala`
  positions).
- Package mode: `goala emit ./...` sibling-file emission,
  `zz_goala_support.go`, cross-file exhaustiveness, `go:generate` docs.
- Negative-test corpus: **every diagnostic has a test asserting message + span**
  (the 3 invalid Phase-0 samples plus ~15 more).
- `bench/` parity suite (fw bench/ pattern): match-vs-switch, Result-vs-(T,error),
  proving zero-runtime claims with numbers.

Gate: full corpus (valid + invalid) green; interop sample builds against a real
third-party Go module; deleting any match arm in any corpus file produces the
exhaustiveness error; bench deltas within noise of hand-written Go.

### Phase 4 — Tooling to the nines (3 weeks)

Deliverables:
- `goala fmt` (idempotent, `--check` for CI) and `goala lint` (≥10 rules).
- `goala lsp` (diagnostics, hover, document symbols, definition) + LSP
  conformance script (fw cmd/ferrous-wheel/lsp_test.go pattern).
- VS Code extension (highlight + LSP client + snippets), packaged `.vsix`.
- canopy integration proof: `canopy index/search refs/graph calls` over
  `demo/project`; upstream registration PR to gotreesitter grammars registry;
  `queries/tags.scm` if canopy's symbol extraction needs it (open question Q6).
- Playground: wasm `goalaTranspile` + CST pane + backend run sandbox, deployed.
- Fuzzing (fw fuzz_test.go pattern: parser never panics, transpiler never emits
  non-gofmt-able output) and race-enabled test runs.

Gate: fmt idempotence fuzz clean; LSP conformance script green; extension
installs and highlights in stock VS Code; canopy refs/callgraph demo checklist
green; playground live.

### Phase 5 — Docs + thesis demo (1–2 weeks)

Deliverables:
- `demo/run.sh` + assets (section 5), Helix/Zed configs, asciinema recording.
- README (fw-completeness: quick taste, install, full feature tour), language
  tour doc, "The thesis" doc adapted from this spec's sections 0–1.
- Release: tagged v0.1, `go install m31labs.dev/goala/cmd/goala@latest`,
  published `.vsix`, playground URL.

Gate: `make demo` runs end-to-end on a clean machine (Go + cc + Neovim +
canopy only); a person who has never seen goala can follow README to a working
`goala run` in under 5 minutes.

**Total: ~11–14 engineering weeks** to the ferrous-wheel completeness bar.
Phases 1–2 (the thesis-critical spine) land in ~5 weeks.

---

## 8. Risks and de-risking

| # | Risk | Severity | De-risk |
|---|---|---|---|
| R1 | Grammar ambiguity spiral (lambda-vs-parens, pattern-vs-call) stalls Phase 1 | High | Corpus-first; `GenerateWithReport` whitelist in CI from day one; copy grammargen's TypeScript resolutions for the arrow ambiguity; patterns are a separate sub-grammar, not expressions. Escape hatch: any arm shape that resists LR gets a keyword disambiguator (`case Circle(r) =>`) — syntax freeze notes this as the fallback. |
| R2 | `parser.c` emit diverges from pure-Go parse for goala's shapes | High (it's the thesis) | The Phase-1 C parity harness runs the *entire corpus* through both runtimes on every CI run; grammargen's existing top-50 parity suite means the emitter is well-trodden — goala only needs to prove its own grammar stays inside emitter-supported territory (no external scanner planned: Go-style explicit semicolon-free lines are avoided by making newlines insignificant and `,`-separated arms explicit — goala needs no `_automatic_semicolon` external, dodging the known `-lr-split`/external interaction documented in grammargen/README.md:102-110). |
| R3 | Exhaustiveness needs type info the local inferencer can't produce | Medium | Designed degradation (2.2): unresolved subject ⇒ require wildcard + explain. Never a soundness hole, at worst a UX nudge to annotate. |
| R4 | `go/packages` interop loading is slow/fragile (no network, vendored deps, wasm playground) | Medium | Module-level signature cache; degradation mode (assume `(T,error)` under `?`); playground wasm ships with a pre-baked stdlib signature index and disables third-party imports. |
| R5 | Type inference scope creep (the fw typeenv grew large) | Medium | Obligations-only checker charter (4.1) enforced in review; the Go compiler owns everything else; any inference feature must cite the obligation it serves. |
| R6 | Interop edge cases: variadics under `?`, multi-value non-error returns, generic Go funcs, dot-import shadowing | Medium | Explicit v1 matrix in docs: `?` supports exactly `(T, error)`/`(T, bool)`/`Result`/`Option` operands; everything else is a compile error with a message, never a guess. Edge-case corpus grows in Phase 3. |
| R7 | Name collisions between injected helpers and user code | Low | fw-proven: usage-gated injection + collision detection (fw README: avoids colliding with user `Result/Ok/...`), `__goala` prefix for internals. |
| R8 | Formatter instability (fmt not idempotent) | Low | Idempotence fuzz gate in Phase 4; formatter is CST-driven (never re-parses its own output differently — same grammar). |
| R9 | canopy indexes `.goala` but extracts no symbols ("tagless grammars are skipped", canopy README) | Medium (hits P2) | Investigate early (Phase 1 registry smoke test includes a canopy index run); if needed, add `tags.scm` support to `ExtensionEntry` or supply supertype metadata canopy's extractor understands — budgeted in Phase 4, flagged as Q6. |
| R10 | Editor-side surprises (Neovim ABI version pinning, Zed wasm grammar build) | Low | ABI 14/15 is what EmitC targets and what current editors consume; demo assets vendored + tested in CI containers for nvim; Helix/Zed checked manually with written checklists. |

---

## 9. Open questions

- **Q1 — Lambda arrow final form.** `(x) => e` / `x => e` (spec'd) vs `|x| e`.
  Decide in Phase 0 syntax freeze; `=>` is spec'd because match arms already
  use it and grammargen has the TS-style conflict resolution ready.
- **Q2 — `struct Name(...)` records vs Go-style struct bodies only.** Spec says
  both (concise records + Go `type` passthrough). If grammar cost of concise
  records exceeds a day of conflict work, drop to Go-style bodies without
  harming the thesis.
- **Q3 — `Option[T]` zero-value semantics for interop.** Struct representation
  means `var o Option[int]` is a valid `None` — good. But should map reads
  auto-lift outside `match` subjects? v1: only in `match`/`?` positions;
  revisit with usage.
- **Q4 — Sibling-file emission vs shadow build dir for package mode.** Sibling
  `_goala.go` files (spec'd) maximize interop and `go:generate` citizenship but
  put generated code in the working tree. Alternative: `.goala-build/` overlay
  with `go build -overlay`. Decide in Phase 3 after CLI ergonomics testing.
- **Q5 — Upstream registration timing.** Registering goala inside
  gotreesitter's grammars registry makes stock canopy work with zero setup but
  couples releases. Propose: after Phase 4 gate, as an `ExtensionEntry`-style
  contribution like fw/danmuji.
- **Q6 — canopy symbol extraction for registered extensions.** Does canopy's
  indexer extract definitions from extension grammars via supertypes/heuristics,
  or does it require per-language tags queries? Determines whether P2 is
  literally zero canopy work or "zero code, one query file". Answer empirically
  in Phase 1's registry smoke test.
- **Q7 — Incremental parsing in the LSP.** gotreesitter's incremental parser is
  under active hardening (gotreesitter branch work, July 2026). v1 LSP does
  full reparses (goala files are small); adopt incremental when the upstream
  correctness work settles.
- **Q8 — `derive Json` (gala's zero-reflection codec) as the v1.1 flagship.**
  The fw derive machinery + goala's sealed metadata make compiler-generated
  marshalers a natural next proof point ("goala gets gala's JSON story for a
  weekend of work"). Out of v1 scope; keep the sealed registry's shape amenable.

---

## 10. Appendix — studied sources (citations)

- ferrous-wheel: `README.md` (completeness bar, CLI, design notes),
  `grammar.go:32` (`ExtendGrammar` usage), `grammar.go:800-818` (extras
  lesson), `grammar.go:824-886` (cost of fighting a base lexer),
  `transpile.go:23-28` (cached Language), `register/register.go`,
  `cmd/ferrous-wheel/main.go` (CLI semantics), `cmd/ferrous-wheel/lsp.go`
  (thin LSP scope), `playground/wasm/main.go:19-44`, `sourcemap.go`,
  `typeenv*.go`, `bench/`, `editor/vscode/`.
- grammargen: `grammar.go:322-379` (`ExtendGrammar` deep-copy),
  `encode.go:29/:35` (`GenerateLanguage`, `GenerateLanguageAndBlob`),
  `codegen_c.go:13-48` (`GenerateC`/`EmitC`, ABI emit),
  `diagnostics.go:742-752/:1085` (`GenerateReport`, `GenerateWithReport`),
  `import_grammarjson.go:132`, `export_grammarjson.go`, `emit_grammar_go.go`,
  `highlight_gen.go`, `typescript_grammar.go` (arrow-ambiguity precedent),
  `README.md` (doctor/parse/emit workflow; `-lr-split` external-scanner caveat
  at lines 102-110).
- gotreesitter grammars registry: `registry.go:153-178`
  (`ExtensionEntry`/`RegisterExtension`), `:249-277` (`DetectLanguage`
  extension map), `:400-419` (alias resolution incl. `fw`).
- canopy: `README.md` (command surface, 206+ languages, tagless-grammar note),
  `cmd/canopyls/main.go` (LSP), `pkg/scope/index.go:17`,
  `pkg/complexity/complexity.go:86`, `internal/scope/scope.go:63`
  (registry-driven detection).
- gala: github.com/martianoff/gala README (sealed types + exhaustive match,
  monad stack, `bind`/`also`, zero-reflection JSON, `(T,error)`→`Try[T]`,
  ANTLR/JVM/Bazel toolchain, `gala mod/run/build/test/lsp`, GoLand plugin).
