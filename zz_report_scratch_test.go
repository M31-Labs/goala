package goala

import (
	"fmt"
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter/grammargen"
)

func TestReportScratch(t *testing.T) {
	rep, err := grammargen.GenerateWithReport(Grammar())
	if err != nil { t.Fatalf("gen: %v", err) }
	ng, _ := grammargen.Normalize(Grammar())
	unres := 0
	for _, c := range rep.Conflicts {
		r := strings.ToLower(c.Resolution)
		ok := strings.Contains(r, " > ") || strings.Contains(r, "default yacc") || strings.Contains(r, "associat") ||
			(strings.Contains(r, "wins (prec ") && !strings.Contains(r, "prec 0)"))
		if !ok { unres++; if unres <= 5 { fmt.Printf("UNRES: %s\n", c.String(ng)) } }
	}
	fmt.Printf("TOTAL=%d UNRESOLVED=%d\n", len(rep.Conflicts), unres)
}

func TestParseSmoke(t *testing.T) {
	srcs := map[string]string{
		"exprfn": "func area(s Shape) float64 = match s {\n  Circle(r) => math.Pi * r * r,\n  Rect(w, h) => w * h,\n  Point => 0,\n}\n",
		"block":  "func demo() {\n  let x = 42\n  var count = 0\n  count = count + 1\n  fmt.Println(x)\n}\n",
		"try":    "func readPort(path string) Result[int] {\n  let data = os.ReadFile(path)?\n  let port = strconv.Atoi(string(data))?\n  Ok(port)\n}\n",
		"fstr":   "func main() {\n  let msg = f\"count=${count}, doubled=${double(count)}\"\n  fmt.Println(msg)\n}\n",
		"forin":  "func tally(tokens []Token) Stats {\n  var s = Stats(0, 0, 0)\n  for t in tokens {\n    match t {\n      Word(_) => { s.words = s.words + 1 },\n      Number(_) => { s.numbers = s.numbers + 1 },\n    }\n  }\n  s\n}\n",
		"iface":  "type Handler interface {\n  Handle(msg string) Result[int]\n}\n",
		"lambda": "func demo() {\n  let double = (n int) => n * 2\n  let inc = n => n + 1\n}\n",
		"guard":  "func classify(raw string) Token = match raw {\n  r if isNumeric(r) => Number(mustAtoi(r)),\n  r => Word(r),\n}\n",
		"comp":   "func load() Result[[]Token] {\n  var tokens = []Token{}\n  tokens = append(tokens, classify(f))\n  Ok(tokens)\n}\n",
		"forcond":"func demo() {\n  for count < 10 {\n    count = count + 1\n  }\n}\n",
		"ifexpr": "func demo() {\n  let label = if count > 0 { \"items\" } else { \"empty\" }\n}\n",
		"import": "package main\n\nimport (\n  \"fmt\"\n  \"os\"\n)\n",
	}
	fails := 0
	for name, s := range srcs {
		sx, _ := ParseSExpr([]byte(s))
		if strings.Contains(sx, "ERROR") || strings.Contains(sx, "MISSING") {
			fails++; fmt.Printf("FAIL %-8s %s\n", name, sx)
		} else { fmt.Printf("OK   %s\n", name) }
	}
	if fails > 0 { t.Fatalf("%d failed", fails) }
}
