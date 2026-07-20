package register_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"

	// Blank import registers goala with the grammars registry via init().
	_ "m31labs.dev/goala/register"
)

// TestDetectLanguageResolvesGoala is the P2 smoke gate: once register is
// imported, the registry must resolve a ".goala" path to goala and hand back a
// working language that parses a corpus file ERROR-free — the whole basis for
// inherited canopy indexing/LSP.
func TestDetectLanguageResolvesGoala(t *testing.T) {
	entry := grammars.DetectLanguage("example.goala")
	if entry == nil {
		t.Fatal("DetectLanguage(\"example.goala\") = nil; registration did not take effect")
	}
	if entry.Name != "goala" {
		t.Fatalf("DetectLanguage resolved to %q, want goala", entry.Name)
	}
	if entry.Language == nil {
		t.Fatal("resolved entry has nil Language loader")
	}
	lang := entry.Language()
	if lang == nil {
		t.Fatal("Language loader returned nil")
	}

	// Parse the anchor corpus program through the registry-resolved language.
	src, err := os.ReadFile(filepath.Join("..", "corpus", "tokens.goala"))
	if err != nil {
		t.Fatalf("read anchor corpus: %v", err)
	}
	tree, err := gotreesitter.NewParser(lang).Parse(src)
	if err != nil {
		t.Fatalf("parse via registry language: %v", err)
	}
	sx := tree.RootNode().SExpr(lang)
	if strings.Contains(sx, "(ERROR") || strings.Contains(sx, "(MISSING") {
		t.Fatalf("registry-resolved parse produced ERROR/MISSING:\n%s", sx)
	}
}

// TestRegistryHighlightQueryPresent verifies the registered entry carries
// goala's highlight query, so registry-driven highlighting has a baseline.
func TestRegistryHighlightQueryPresent(t *testing.T) {
	entry := grammars.DetectLanguage("x.goala")
	if entry == nil {
		t.Fatal("DetectLanguage returned nil")
	}
	if strings.TrimSpace(entry.HighlightQuery) == "" {
		t.Fatal("registered goala entry has empty HighlightQuery")
	}
}
