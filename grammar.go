package goala

// grammar.go is THE goala grammar — the single source of truth for every
// downstream artifact (pure-Go parser, parser.c, grammar.json, blob, highlight
// query). It is authored with grammargen's Go DSL as a DISTINCT grammar
// (NewGrammar("goala")), not an extension of Go's grammar: the goala spine
// (sealed types, match-expressions, `?` propagation, lambdas, f-strings) is
// written fresh from the start rule up, and only Go's *lexical* sub-grammars
// (identifiers, comments, numeric/string/rune literals) plus package/import
// clauses are BORROWED verbatim via ImportRules. This proves grammargen is a
// language-authoring tool, not merely a Go-superset tool.
//
// Keyword set (extracted because SetWord("identifier") is set):
//   package import func sealed struct type interface let var bind match if
//   else for in return break continue derive map chan func
//
// The first Define below (source_file) is the start rule.
func Grammar() *GrammarType {
	g := NewGrammar("goala")

	// -------------------------------------------------------------------------
	// Structure
	// -------------------------------------------------------------------------

	// source_file is the start rule (first Define wins in grammargen).
	//
	// Top-level items carry an optional trailing `\n`/`;` separator, exactly as
	// block statements do. This is load-bearing: because `\n` is a live token
	// (the statement separator), a function body leaves the parser in a state
	// whose follow-set includes `\n`, so a trailing newline after a declaration
	// must be absorbable here or it lexes as a stray token and errors. Go's
	// grammar separates top-level declarations the same way.
	g.Define("source_file", Seq(
		Optional(Seq(Sym("package_clause"), Optional(Sym("_statement_separator")))),
		Repeat(Seq(Sym("import_declaration"), Optional(Sym("_statement_separator")))),
		Repeat(Seq(Sym("_declaration"), Optional(Sym("_statement_separator")))),
	))

	g.Define("_declaration", Choice(
		Sym("sealed_declaration"),
		Sym("struct_declaration"),
		Sym("function_declaration"),
		Sym("derive_declaration"),
		Sym("go_type_declaration"),
	))

	// sealed Shape { Circle(radius float64), Rect(width float64, height float64), Point }
	g.Define("sealed_declaration", Seq(
		Str("sealed"),
		Field("name", Sym("identifier")),
		Optional(Field("type_parameters", Sym("type_parameters"))),
		Str("{"),
		CommaSep1(Sym("sealed_case")),
		Optional(Str(",")),
		Str("}"),
	))

	// A sealed case is either a record (Circle(radius float64)) or nullary (Point).
	g.Define("sealed_case", Choice(
		Seq(
			Field("name", Sym("identifier")),
			Str("("),
			CommaSep1(Sym("field_specification")),
			Optional(Str(",")),
			Str(")"),
		),
		Field("name", Sym("identifier")),
	))

	// A named field: `radius float64`.
	g.Define("field_specification", Seq(
		Field("name", Sym("identifier")),
		Field("type", Sym("_type")),
	))

	// struct Vec(x float64, y float64)
	g.Define("struct_declaration", Seq(
		Str("struct"),
		Field("name", Sym("identifier")),
		Optional(Field("type_parameters", Sym("type_parameters"))),
		Str("("),
		CommaSep1(Sym("field_specification")),
		Optional(Str(",")),
		Str(")"),
	))

	// func area(s Shape) float64 = match s { ... }
	// func (v Vec) norm() float64 = math.Sqrt(...)
	// func demo() { ... }
	g.Define("function_declaration", Seq(
		Str("func"),
		Optional(Field("receiver", Sym("method_receiver"))),
		Field("name", Sym("identifier")),
		Optional(Field("type_parameters", Sym("type_parameters"))),
		Field("parameters", Sym("parameter_list")),
		Optional(Field("result", Sym("_type"))),
		Field("body", Sym("_function_body")),
	))

	g.Define("_function_body", Choice(
		Sym("block"),
		Seq(Str("="), Field("value", Sym("_expression"))),
	))

	g.Define("method_receiver", Seq(
		Str("("),
		Field("name", Sym("identifier")),
		Field("type", Sym("_type")),
		Str(")"),
	))

	g.Define("parameter_list", Seq(
		Str("("),
		Optional(Seq(
			CommaSep1(Sym("parameter")),
			Optional(Str(",")),
		)),
		Str(")"),
	))

	g.Define("parameter", Seq(
		Field("name", Sym("identifier")),
		Field("type", Sym("_type")),
	))

	// derive Stringer for Shape
	g.Define("derive_declaration", Seq(
		Str("derive"),
		Field("trait", Sym("identifier")),
		Str("for"),
		Field("type", Sym("identifier")),
	))

	// type Meters = float64
	// type Handler interface { Handle(msg string) Result[int] }
	g.Define("go_type_declaration", Seq(
		Str("type"),
		Field("name", Sym("identifier")),
		Optional(Str("=")),
		Field("type", Sym("_type")),
	))

	// -------------------------------------------------------------------------
	// Types (authored fresh; mirror Go's type syntax so any Go type is nameable)
	// -------------------------------------------------------------------------

	g.Define("_type", Choice(
		Sym("type_identifier"),
		Sym("qualified_type"),
		Sym("generic_type"),
		Sym("pointer_type"),
		Sym("slice_type"),
		Sym("map_type"),
		Sym("channel_type"),
		Sym("function_type"),
		Sym("interface_type"),
	))

	g.Define("type_identifier", Alias(Sym("identifier"), "type_identifier", true))

	// pkg.Name
	g.Define("qualified_type", Seq(
		Field("package", Sym("identifier")),
		Str("."),
		Field("name", Sym("type_identifier")),
	))

	// Name[T], Result[int], Option[[]Token]
	g.Define("generic_type", Prec(1, Seq(
		Field("type", Choice(Sym("type_identifier"), Sym("qualified_type"))),
		Field("type_arguments", Sym("type_arguments")),
	)))

	g.Define("type_arguments", Seq(
		Str("["),
		CommaSep1(Sym("_type")),
		Optional(Str(",")),
		Str("]"),
	))

	g.Define("pointer_type", Seq(Str("*"), Sym("_type")))

	g.Define("slice_type", Seq(Str("["), Str("]"), Field("element", Sym("_type"))))

	g.Define("map_type", Seq(
		Str("map"),
		Str("["),
		Field("key", Sym("_type")),
		Str("]"),
		Field("value", Sym("_type")),
	))

	g.Define("channel_type", Seq(Str("chan"), Sym("_type")))

	g.Define("function_type", Seq(
		Str("func"),
		Str("("),
		Optional(Seq(CommaSep1(Sym("_type")), Optional(Str(",")))),
		Str(")"),
		Optional(Field("result", Sym("_type"))),
	))

	g.Define("interface_type", Seq(
		Str("interface"),
		Str("{"),
		Optional(Seq(
			Sym("method_specification"),
			Repeat(Seq(Sym("_statement_separator"), Sym("method_specification"))),
			Optional(Sym("_statement_separator")),
		)),
		Str("}"),
	))

	g.Define("method_specification", Seq(
		Field("name", Sym("identifier")),
		Field("parameters", Sym("parameter_list")),
		Optional(Field("result", Sym("_type"))),
	))

	// Generic type parameters: [T any], [T any, U comparable]
	g.Define("type_parameters", Seq(
		Str("["),
		CommaSep1(Sym("type_parameter")),
		Optional(Str(",")),
		Str("]"),
	))

	g.Define("type_parameter", Seq(
		Field("name", Sym("identifier")),
		Field("constraint", Sym("_type")),
	))

	// -------------------------------------------------------------------------
	// Statements
	// -------------------------------------------------------------------------

	g.Define("block", Seq(
		Str("{"),
		Optional(Sym("_statement_list")),
		Str("}"),
	))

	// Statements are separated by a newline or an explicit semicolon — the Go
	// ASI idiom encoded without an external scanner. `\n` is dual-role: skipped
	// as an extra inside expressions / multi-line call args, but consumed as a
	// separator here where the parse state expects one. This is what keeps
	// adjacent bare-expression statements (`a` then `-b`) from being read as one
	// binary expression `a - b`; the calc grammar's doc comment (grammargen
	// calc_grammar.go) names this exact ambiguity as the reason it forbids
	// statement lists — goala resolves it with the separator instead.
	g.Define("_statement_list", Seq(
		Sym("_statement"),
		Repeat(Seq(Sym("_statement_separator"), Sym("_statement"))),
		Optional(Sym("_statement_separator")),
	))

	g.Define("_statement_separator", Choice(Pat(`\n`), Str(";")))

	g.Define("_statement", Choice(
		Sym("let_declaration"),
		Sym("var_declaration"),
		Sym("bind_declaration"),
		Sym("return_statement"),
		Sym("break_statement"),
		Sym("continue_statement"),
		Sym("for_in_statement"),
		Sym("for_statement"),
		Sym("assignment"),
		Sym("expression_statement"),
	))

	// let x = 42  (immutable)
	g.Define("let_declaration", Seq(
		Str("let"),
		Field("name", Sym("identifier")),
		Optional(Field("type", Sym("_type"))),
		Str("="),
		Field("value", Sym("_expression")),
	))

	// var count = 0  (mutable)
	g.Define("var_declaration", Seq(
		Str("var"),
		Field("name", Sym("identifier")),
		Optional(Field("type", Sym("_type"))),
		Str("="),
		Field("value", Sym("_expression")),
	))

	// bind order = fetchOrder(id)   == let order = fetchOrder(id)?
	g.Define("bind_declaration", Seq(
		Str("bind"),
		Field("name", Sym("identifier")),
		Str("="),
		Field("value", Sym("_expression")),
	))

	// count = count + 1
	g.Define("assignment", Prec(1, Seq(
		Field("left", Sym("_expression")),
		Str("="),
		Field("right", Sym("_expression")),
	)))

	g.Define("return_statement", Seq(
		Str("return"),
		Optional(Field("value", Sym("_expression"))),
	))

	g.Define("break_statement", Str("break"))
	g.Define("continue_statement", Str("continue"))

	// for s in shapes { ... }   /   for i, s in shapes { ... }
	g.Define("for_in_statement", Seq(
		Str("for"),
		Field("left", Sym("_range_binding")),
		Str("in"),
		Field("range", Sym("_expression")),
		Field("body", Sym("block")),
	))

	g.Define("_range_binding", Choice(
		Sym("identifier"),
		Seq(Sym("identifier"), Str(","), Sym("identifier")),
	))

	// for count < 10 { ... }  (condition-for)
	g.Define("for_statement", Seq(
		Str("for"),
		Field("condition", Sym("_expression")),
		Field("body", Sym("block")),
	))

	g.Define("expression_statement", Sym("_expression"))

	// -------------------------------------------------------------------------
	// Expressions
	// -------------------------------------------------------------------------

	g.Define("_expression", Choice(
		Sym("identifier"),
		Sym("_literal"),
		Sym("interpolated_string"),
		Sym("call_expression"),
		Sym("selector_expression"),
		Sym("index_expression"),
		Sym("try_expression"),
		Sym("unary_expression"),
		Sym("binary_expression"),
		Sym("parenthesized_expression"),
		Sym("composite_literal"),
		Sym("match_expression"),
		Sym("if_expression"),
		Sym("lambda_expression"),
	))

	g.Define("_literal", Choice(
		Sym("int_literal"),
		Sym("float_literal"),
		Sym("interpreted_string_literal"),
		Sym("raw_string_literal"),
		Sym("rune_literal"),
	))

	g.Define("call_expression", Prec(11, Seq(
		Field("function", Sym("_expression")),
		Field("arguments", Sym("argument_list")),
	)))

	g.Define("argument_list", Seq(
		Str("("),
		Optional(Seq(
			CommaSep1(Sym("_expression")),
			Optional(Str(",")),
		)),
		Str(")"),
	))

	g.Define("selector_expression", Prec(11, Seq(
		Field("operand", Sym("_expression")),
		Str("."),
		Field("field", Sym("identifier")),
	)))

	g.Define("index_expression", Prec(11, Seq(
		Field("operand", Sym("_expression")),
		Str("["),
		Field("index", Sym("_expression")),
		Str("]"),
	)))

	// postfix `?` error propagation
	g.Define("try_expression", Prec(12, Seq(
		Field("operand", Sym("_expression")),
		Str("?"),
	)))

	g.Define("unary_expression", PrecRight(9, Seq(
		Field("operator", Choice(Str("!"), Str("-"), Str("+"))),
		Field("operand", Sym("_expression")),
	)))

	g.Define("binary_expression", Choice(
		PrecLeft(8, Seq(Field("left", Sym("_expression")), Field("operator", Choice(Str("*"), Str("/"), Str("%"), Str("<<"), Str(">>"), Str("&"))), Field("right", Sym("_expression")))),
		PrecLeft(7, Seq(Field("left", Sym("_expression")), Field("operator", Choice(Str("+"), Str("-"), Str("|"), Str("^"))), Field("right", Sym("_expression")))),
		PrecLeft(6, Seq(Field("left", Sym("_expression")), Field("operator", Choice(Str("=="), Str("!="), Str("<="), Str(">="), Str("<"), Str(">"))), Field("right", Sym("_expression")))),
		PrecLeft(5, Seq(Field("left", Sym("_expression")), Field("operator", Str("&&")), Field("right", Sym("_expression")))),
		PrecLeft(4, Seq(Field("left", Sym("_expression")), Field("operator", Str("||")), Field("right", Sym("_expression")))),
	))

	g.Define("parenthesized_expression", Seq(
		Str("("),
		Sym("_expression"),
		Str(")"),
	))

	// []Token{} / []Token{a, b} / map[string]int{}
	g.Define("composite_literal", Seq(
		Field("type", Choice(Sym("slice_type"), Sym("map_type"))),
		Field("body", Sym("literal_value")),
	))

	g.Define("literal_value", Seq(
		Str("{"),
		Optional(Seq(
			CommaSep1(Sym("_expression")),
			Optional(Str(",")),
		)),
		Str("}"),
	))

	// match subj { pat [guard] => body, ... }
	g.Define("match_expression", Seq(
		Str("match"),
		Field("subject", Sym("_expression")),
		Str("{"),
		CommaSep1(Sym("match_arm")),
		Optional(Str(",")),
		Str("}"),
	))

	g.Define("match_arm", Seq(
		Field("pattern", Sym("_pattern")),
		Optional(Field("guard", Sym("guard"))),
		Str("=>"),
		Field("body", Choice(Sym("_expression"), Sym("block"))),
	))

	g.Define("guard", Seq(
		Str("if"),
		Field("condition", Sym("_expression")),
	))

	// if cond { ... } else { ... }  (expression)
	g.Define("if_expression", Seq(
		Str("if"),
		Field("condition", Sym("_expression")),
		Field("consequence", Sym("block")),
		Optional(Seq(
			Str("else"),
			Field("alternative", Choice(Sym("block"), Sym("if_expression"))),
		)),
	))

	// (n int) => n * 2   /   n => n + 1
	//
	// Parenthesized lambda parameters carry explicit types (`(n int)`), which is
	// never a valid parenthesized expression (`n int` is two identifiers), so the
	// classic `(x) => x` vs `(x)` ambiguity never arises. A bare single untyped
	// parameter uses the `n =>` form; the `=>` lookahead resolves it by default
	// shift. This is the deliberate syntax-freeze choice (SYNTAX.md Q1).
	g.Define("lambda_expression", Choice(
		Seq(
			Field("parameters", Sym("lambda_parameters")),
			Str("=>"),
			Field("body", Sym("_expression")),
		),
		Seq(
			Field("parameter", Sym("identifier")),
			Str("=>"),
			Field("body", Sym("_expression")),
		),
	))

	g.Define("lambda_parameters", Seq(
		Str("("),
		Optional(Seq(
			CommaSep1(Sym("lambda_parameter")),
			Optional(Str(",")),
		)),
		Str(")"),
	))

	g.Define("lambda_parameter", Seq(
		Field("name", Sym("identifier")),
		Field("type", Sym("_type")),
	))

	// f-string: f"count=${count}, doubled=${double(count)}"
	g.Define("interpolated_string", Seq(
		Token(Str(`f"`)),
		Repeat(Choice(
			Sym("interpolation"),
			Sym("_string_fragment"),
		)),
		Str(`"`),
	))

	g.Define("interpolation", Seq(
		Str("${"),
		Field("expression", Sym("_expression")),
		Str("}"),
	))

	g.Define("_string_fragment", Alias(Token(Pat(`[^"$]+`)), "string_fragment", true))

	// -------------------------------------------------------------------------
	// Patterns (a separate sub-grammar reachable only from match_arm — patterns
	// are NOT expressions, which is what keeps `Circle(r)` from parsing as a
	// call inside match arms). A bare identifier pattern is a binder; the
	// semantic layer reclassifies capitalized nullary binders (Point/None) as
	// nullary constructors, so constructor_pattern here always carries `(...)`.
	// -------------------------------------------------------------------------

	g.Define("_pattern", Choice(
		Sym("constructor_pattern"),
		Sym("literal_pattern"),
		Sym("binder_pattern"),
		Sym("wildcard_pattern"),
	))

	g.Define("constructor_pattern", Seq(
		Field("constructor", Sym("identifier")),
		Str("("),
		Optional(Seq(
			CommaSep1(Sym("_binding_pattern")),
			Optional(Str(",")),
		)),
		Str(")"),
	))

	g.Define("_binding_pattern", Choice(
		Sym("binder_pattern"),
		Sym("wildcard_pattern"),
	))

	g.Define("binder_pattern", Field("name", Sym("identifier")))
	g.Define("wildcard_pattern", Str("_"))

	g.Define("literal_pattern", Choice(
		Sym("int_literal"),
		Sym("float_literal"),
		Sym("interpreted_string_literal"),
		Sym("raw_string_literal"),
		Sym("rune_literal"),
	))

	// -------------------------------------------------------------------------
	// Borrowed lexical sub-grammars from Go (the interop guarantee, cheaply).
	// ImportRules deep-clones each root + its transitive Sym() closure, stopping
	// at any symbol goala already defines above.
	// -------------------------------------------------------------------------
	ImportRules(g, GoGrammar(),
		"identifier",
		"comment",
		"int_literal",
		"float_literal",
		"rune_literal",
		"interpreted_string_literal",
		"raw_string_literal",
		"package_clause",
		"import_declaration",
	)

	// -------------------------------------------------------------------------
	// Grammar-level settings
	// -------------------------------------------------------------------------

	// Keyword extraction off the identifier token.
	g.SetWord("identifier")

	// Extras restricted to Go's real whitespace (never \v/\f) plus comments —
	// the ferrous-wheel lesson (grammar.go:800-818): so verbatim source echoes
	// can never smuggle exotic whitespace into emitted Go.
	g.SetExtras(
		Sym("comment"),
		Pat(`[ \t\r\n]`),
	)

	// Supertype metadata for richer queries / canopy code-intel.
	g.SetSupertypes(
		"_expression",
		"_statement",
		"_declaration",
		"_pattern",
		"_type",
	)

	// Hidden helper nonterminals inlined so they don't clutter the CST.
	g.SetInline(
		"_literal",
		"_function_body",
		"_range_binding",
		"_binding_pattern",
		"_statement_separator",
		// Carried over from Go's inline set: these borrowed hidden rules MUST be
		// inlined (as Go inlines them) or the parser sees both `import_spec → X`
		// and `_string_literal → X` reductions, an 89-way reduce/reduce artifact.
		"_string_literal",
		"_package_identifier",
	)

	return g
}
