package goala

import (
	"strings"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

// TestHighlightQueryCompiles is the Phase-1 highlight gate: the hand-tuned
// highlights.scm must compile against goala's language under gotreesitter's
// query engine. A capture referencing a node type or field the grammar does not
// define would fail here.
func TestHighlightQueryCompiles(t *testing.T) {
	q := HighlightQuery()
	if strings.TrimSpace(q) == "" {
		t.Fatal("HighlightQuery is empty")
	}
	lang, err := Language()
	if err != nil {
		t.Fatalf("Language: %v", err)
	}
	query, err := gotreesitter.NewQuery(q, lang)
	if err != nil {
		t.Fatalf("highlights.scm does not compile under the query engine: %v", err)
	}
	_ = query
}
