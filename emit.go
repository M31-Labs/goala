package goala

import (
	_ "embed"
	"fmt"

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

//go:embed queries/highlights.scm
var highlightQuerySource string

// HighlightQuery returns goala's tree-sitter highlight query — the hand-tuned
// queries/highlights.scm, embedded at build time. This is the value register/
// hands to the grammars registry and the file `goala grammar emit -highlight`
// writes. It is authored by hand because grammargen's highlight inference is a
// diff over a base grammar and produces nothing for a from-scratch grammar (see
// InferredHighlightQuery).
func HighlightQuery() string {
	return highlightQuerySource
}

// InferredHighlightQuery returns grammargen's automatically inferred highlight
// query. For goala it is empty: GenerateHighlightQueries derives captures from
// the rules an EXTENDED grammar adds over its base, and goala is authored from
// NewGrammar up (no base to diff against). Kept for completeness and to document
// the limitation that drives the hand-authored HighlightQuery above.
func InferredHighlightQuery() string {
	g := Grammar()
	return grammargen.GenerateHighlightQueries(g, g)
}
