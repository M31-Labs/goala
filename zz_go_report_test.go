package goala

import (
	"fmt"
	"testing"

	"github.com/odvcencio/gotreesitter/grammargen"
)

func TestGoGrammarReport(t *testing.T) {
	g := GoGrammar()
	rep, err := grammargen.GenerateWithReport(g)
	if err != nil { t.Fatalf("gen: %v", err) }
	sr, rr := 0, 0
	for _, c := range rep.Conflicts {
		if c.Kind == grammargen.ShiftReduce { sr++ } else { rr++ }
	}
	fmt.Printf("GO: symbols=%d states=%d conflicts=%d (SR=%d RR=%d)\n", rep.SymbolCount, rep.StateCount, len(rep.Conflicts), sr, rr)
	fmt.Printf("ShiftReduce const=%d ReduceReduce const=%d\n", grammargen.ShiftReduce, grammargen.ReduceReduce)
}
