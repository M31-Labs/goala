package goala

// Re-export grammargen types and DSL functions for use in the goala grammar
// definition. This keeps grammar.go readable without dot-imports.
//
// The Grammar type is aliased as GrammarType so the package-level Grammar()
// function (which returns the goala grammar) does not collide with the type.

import (
	"github.com/odvcencio/gotreesitter/grammargen"
)

// Type aliases.
type (
	GrammarType = grammargen.Grammar
	Rule        = grammargen.Rule
)

// Constructor aliases.
var (
	NewGrammar    = grammargen.NewGrammar
	ExtendGrammar = grammargen.ExtendGrammar
)

// DSL function aliases.
var (
	Str         = grammargen.Str
	Pat         = grammargen.Pat
	Sym         = grammargen.Sym
	Seq         = grammargen.Seq
	Choice      = grammargen.Choice
	Repeat      = grammargen.Repeat
	Repeat1     = grammargen.Repeat1
	Optional    = grammargen.Optional
	Token       = grammargen.Token
	ImmToken    = grammargen.ImmToken
	Field       = grammargen.Field
	Prec        = grammargen.Prec
	PrecLeft    = grammargen.PrecLeft
	PrecRight   = grammargen.PrecRight
	PrecDynamic = grammargen.PrecDynamic
	Alias       = grammargen.Alias
	Blank       = grammargen.Blank
	CommaSep    = grammargen.CommaSep
	CommaSep1   = grammargen.CommaSep1
)

// Helper aliases.
var (
	AppendChoice     = grammargen.AppendChoice
	AddConflict      = grammargen.AddConflict
	GenerateLanguage = grammargen.GenerateLanguage
)

// Emit/export aliases.
var (
	GenerateHighlightQueries = grammargen.GenerateHighlightQueries
	ExportGrammarJSON        = grammargen.ExportGrammarJSON
	ImportGrammarJSON        = grammargen.ImportGrammarJSON
	EmitGrammarGo            = grammargen.EmitGrammarGo
)
