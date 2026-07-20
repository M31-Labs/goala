package goala

import (
	"fmt"
	"sync"

	"github.com/odvcencio/gotreesitter/grammargen"
)

// This file projects the single Grammar() value onto every downstream artifact
// (DESIGN §3.3). Each function is a thin, named wrapper over a grammargen emit
// entry point so the CLI (cmd/goala) and the registry entry (register/) share
// exactly one path from grammar to output — there is no second source of truth.

// EmitParserC returns the tree-sitter C parser source (parser.c, ABI 14/15)
// generated from the goala grammar. This is the thesis centerpiece: the same
// grammar that drives the pure-Go parser emits a standard parser.c that any
// tree-sitter editor loads natively.
func EmitParserC() (string, error) {
	code, err := grammargen.GenerateC(Grammar())
	if err != nil {
		return "", fmt.Errorf("goala: emit parser.c: %w", err)
	}
	return code, nil
}

// EmitBlob returns the serialized gotreesitter grammar blob (goala.bin). This is
// the embeddable/registry form of the language.
func EmitBlob() ([]byte, error) {
	_, blob, err := grammargen.GenerateLanguageAndBlob(Grammar())
	if err != nil {
		return nil, fmt.Errorf("goala: emit blob: %w", err)
	}
	return blob, nil
}

// EmitGrammarJSON returns the resolved tree-sitter grammar.json for the goala
// grammar — the interchange artifact reviewers and the upstream tree-sitter
// ecosystem can read.
func EmitGrammarJSON() ([]byte, error) {
	data, err := grammargen.ExportGrammarJSON(Grammar())
	if err != nil {
		return nil, fmt.Errorf("goala: emit grammar.json: %w", err)
	}
	return data, nil
}

// EmitGrammarGoSource returns the goala grammar re-emitted as Go DSL source (the
// resolved, round-tripped form) — a review artifact proving the grammar
// survives an export/import cycle.
func EmitGrammarGoSource() ([]byte, error) {
	src, err := grammargen.EmitGrammarGo(Grammar(), "goala", "Grammar")
	if err != nil {
		return nil, fmt.Errorf("goala: emit grammar Go DSL: %w", err)
	}
	return src, nil
}

var (
	highlightOnce  sync.Once
	highlightQuery string
)

// HighlightQuery returns the inferred tree-sitter highlight query for goala,
// computed once from the grammar. It is the baseline the tuned
// queries/highlights.scm starts from and the value register/ hands to the
// grammars registry.
func HighlightQuery() string {
	highlightOnce.Do(func() {
		g := Grammar()
		highlightQuery = grammargen.GenerateHighlightQueries(g, g)
	})
	return highlightQuery
}
