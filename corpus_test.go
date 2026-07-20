package goala

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// updateGoldens regenerates the corpus/*.sexpr golden snapshots instead of
// asserting against them. Run: go test -run TestCorpusGoldenParse -update-goldens
var updateGoldens = flag.Bool("update-goldens", false, "rewrite corpus/*.sexpr golden parse snapshots")

// corpusFiles returns the sorted list of .goala programs in the corpus.
func corpusFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("corpus", "*.goala"))
	if err != nil {
		t.Fatalf("glob corpus: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no corpus files found")
	}
	return files
}

// TestCorpusParsesErrorFree is the Phase-1 gate: every corpus program must parse
// through the pure-Go parser with no ERROR or MISSING node.
func TestCorpusParsesErrorFree(t *testing.T) {
	for _, f := range corpusFiles(t) {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			sx, err := ParseSExpr(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if strings.Contains(sx, "(ERROR") || strings.Contains(sx, "(MISSING") {
				t.Fatalf("parse produced ERROR/MISSING node:\n%s", PrettySExpr(sx))
			}
		})
	}
}

// TestCorpusGoldenParse asserts each corpus program's S-expression parse matches
// its committed golden snapshot. This is the syntax-freeze contract: any grammar
// change that alters a corpus parse must be reviewed as a golden diff.
func TestCorpusGoldenParse(t *testing.T) {
	for _, f := range corpusFiles(t) {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			pretty, err := PrettyParse(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			golden := strings.TrimSuffix(f, ".goala") + ".sexpr"
			if *updateGoldens {
				if err := os.WriteFile(golden, []byte(pretty+"\n"), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run -update-goldens to create): %v", err)
			}
			got := pretty + "\n"
			if got != string(want) {
				t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s",
					filepath.Base(f), got, string(want))
			}
		})
	}
}
