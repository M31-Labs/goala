package goala

import (
	"sort"
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter/grammargen"
)

// glrKeptResolution is the exact Resolution string grammargen stamps on a
// conflict it could NOT resolve deterministically and therefore hands to the
// GLR runtime (diagnostics.go). Any such conflict is a genuine grammar
// ambiguity the author must acknowledge — the thing DESIGN §3.2's whitelist
// exists to gate.
const glrKeptResolution = "GLR (multiple actions kept)"

// isDeterministicResolution reports whether a conflict was resolved statically
// by the LR generator (precedence, associativity, or the default shift-wins
// yacc rule) rather than deferred to GLR. These are the generator doing its
// normal job; they need no whitelist.
func isDeterministicResolution(r string) bool {
	return strings.HasPrefix(r, "shift wins") ||
		strings.HasPrefix(r, "reduce wins") ||
		strings.HasPrefix(r, "error (")
}

// TestGrammarGenerates is the smoke gate: the goala grammar compiles to a
// pure-Go Language with no error. If this fails, nothing downstream can run.
func TestGrammarGenerates(t *testing.T) {
	if _, err := grammargen.GenerateLanguage(Grammar()); err != nil {
		t.Fatalf("GenerateLanguage: %v", err)
	}
}

// TestGrammarConflictWhitelist is the Phase-1 conflict gate (DESIGN §3.2).
//
// The goala grammar is authored to be fully LR-deterministic: every
// shift/reduce and reduce/reduce conflict is resolved statically by the
// operator-precedence table (binary_expression PrecLeft levels, postfix `?`,
// selector/index/call precedence) or by the default shift preference
// (if/else, block-vs-expression). ZERO conflicts are handed to the GLR runtime.
//
// This is stronger than a commented whitelist: rather than enumerating tolerated
// ambiguities, the gate asserts there are none. A grammar edit that introduces a
// real ambiguity surfaces here as a GLR-kept conflict and fails CI, which is the
// exact "new unexplained ConflictDiag fails CI" contract DESIGN calls for.
func TestGrammarConflictWhitelist(t *testing.T) {
	rep, err := grammargen.GenerateWithReport(Grammar())
	if err != nil {
		t.Fatalf("GenerateWithReport: %v", err)
	}
	ng, _ := grammargen.Normalize(Grammar())

	var glrKept []grammargen.ConflictDiag
	unknown := map[string]int{}
	for _, c := range rep.Conflicts {
		switch {
		case c.Resolution == glrKeptResolution:
			glrKept = append(glrKept, c)
		case !isDeterministicResolution(c.Resolution):
			unknown[c.Resolution]++
		}
	}

	// Gate 1: no genuine ambiguity kept for the GLR runtime.
	if len(glrKept) != 0 {
		var b strings.Builder
		for i, c := range glrKept {
			if i >= 5 {
				b.WriteString("  ...\n")
				break
			}
			b.WriteString("  " + c.String(ng) + "\n")
		}
		t.Fatalf("%d conflict(s) kept for GLR (grammar is no longer LR-deterministic); "+
			"add an explicit precedence/conflict resolution or whitelist with rationale:\n%s",
			len(glrKept), b.String())
	}

	// Gate 2: every remaining conflict was resolved by a recognized
	// deterministic rule. An unrecognized Resolution string means grammargen
	// grew a new resolution class this gate has not been taught to reason about.
	if len(unknown) != 0 {
		var keys []string
		for k := range unknown {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for _, k := range keys {
			b.WriteString("  " + k + "\n")
		}
		t.Fatalf("unrecognized conflict resolution class(es):\n%s", b.String())
	}

	t.Logf("conflicts=%d all resolved deterministically (0 kept for GLR); symbols=%d states=%d",
		len(rep.Conflicts), rep.SymbolCount, rep.StateCount)
}

// TestFeatureSnippetsParse checks each 2.2 language feature in isolation parses
// ERROR-free. These are faster, more targeted than the full corpus and pinpoint
// which construct broke when a grammar edit regresses.
func TestFeatureSnippetsParse(t *testing.T) {
	snippets := map[string]string{
		"expr_body_match": "func area(s Shape) float64 = match s {\n  Circle(r) => math.Pi * r * r,\n  Rect(w, h) => w * h,\n  Point => 0,\n}\n",
		"block_let_var":   "func demo() {\n  let x = 42\n  var count = 0\n  count = count + 1\n  fmt.Println(x)\n}\n",
		"try_propagation": "func readPort(path string) Result[int] {\n  let data = os.ReadFile(path)?\n  let port = strconv.Atoi(string(data))?\n  Ok(port)\n}\n",
		"f_string":        "func main() {\n  let msg = f\"count=${count}, doubled=${double(count)}\"\n  fmt.Println(msg)\n}\n",
		"for_in_match":    "func tally(tokens []Token) Stats {\n  var s = Stats(0, 0, 0)\n  for t in tokens {\n    match t {\n      Word(_) => { s.words = s.words + 1 },\n      Number(_) => { s.numbers = s.numbers + 1 },\n    }\n  }\n  s\n}\n",
		"interface_type":  "type Handler interface {\n  Handle(msg string) Result[int]\n}\n",
		"lambda_forms":    "func demo() {\n  let double = (n int) => n * 2\n  let inc = n => n + 1\n}\n",
		"guarded_arm":     "func classify(raw string) Token = match raw {\n  r if isNumeric(r) => Number(mustAtoi(r)),\n  r => Word(r),\n}\n",
		"composite_lit":   "func load() Result[[]Token] {\n  var tokens = []Token{}\n  tokens = append(tokens, classify(f))\n  Ok(tokens)\n}\n",
		"for_condition":   "func demo() {\n  for count < 10 {\n    count = count + 1\n  }\n}\n",
		"if_expression":   "func demo() {\n  let label = if count > 0 { \"items\" } else { \"empty\" }\n}\n",
		"import_block":    "package main\n\nimport (\n  \"fmt\"\n  \"os\"\n)\n",
		"sealed_generic":  "sealed Tree[T any] {\n  Leaf(value T),\n  Node(left Tree[T], right Tree[T]),\n}\n",
		"bind_do":         "func f(id int) Result[int] {\n  bind order = fetchOrder(id)\n  Ok(order)\n}\n",
		"derive_decl":     "derive Stringer for Shape\n",
	}
	for name, src := range snippets {
		name, src := name, src
		t.Run(name, func(t *testing.T) {
			sx, err := ParseSExpr([]byte(src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if strings.Contains(sx, "(ERROR") || strings.Contains(sx, "(MISSING") {
				t.Fatalf("parse produced ERROR/MISSING:\n%s", PrettySExpr(sx))
			}
		})
	}
}
