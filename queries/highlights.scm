; Tree-sitter highlight query for goala.
;
; grammargen's highlight inference (GenerateHighlightQueries) is defined as a
; DIFF over a base grammar and yields nothing for a from-scratch NewGrammar, so
; this query is authored by hand against goala's node types. It is the single
; highlight source of truth: embedded into the goala package (HighlightQuery),
; shipped in the registry entry (register/), and emitted verbatim by
; `goala grammar emit -highlight`.

; Keywords
[
  "sealed"
  "struct"
  "func"
  "derive"
  "type"
  "interface"
  "let"
  "var"
  "bind"
  "match"
  "if"
  "else"
  "for"
  "in"
  "return"
  "break"
  "continue"
  "package"
  "import"
  "map"
  "chan"
] @keyword

"for" @keyword.repeat
"return" @keyword.return

; Operators and punctuation
[
  "="
  "=>"
  "?"
  "=="
  "!="
  "<"
  "<="
  ">"
  ">="
  "&&"
  "||"
  "!"
  "+"
  "-"
  "*"
  "/"
  "%"
] @operator

; Comments
(comment) @comment

; Literals
(int_literal) @number
(float_literal) @number.float
(interpreted_string_literal) @string
(raw_string_literal) @string
(rune_literal) @character
(interpolated_string) @string
(string_fragment) @string
(interpolation) @none

; Types
(type_identifier) @type
(qualified_type package: (identifier) @module)

; Declarations
(sealed_declaration name: (identifier) @type.definition)
(struct_declaration name: (identifier) @type.definition)
(function_declaration name: (identifier) @function)
(derive_declaration trait: (identifier) @type)

; Sealed cases are the constructors of the sum type
(sealed_case name: (identifier) @constructor)
(constructor_pattern constructor: (identifier) @constructor)

; Parameters and record fields
(parameter name: (identifier) @variable.parameter)
(lambda_parameter name: (identifier) @variable.parameter)
(field_specification name: (identifier) @property)

; Bindings
(let_declaration name: (identifier) @variable)
(var_declaration name: (identifier) @variable)
(bind_declaration name: (identifier) @variable)

; Patterns
(binder_pattern name: (identifier) @variable)
(wildcard_pattern) @variable.builtin

; Calls and selectors
(call_expression function: (identifier) @function.call)
(selector_expression field: (identifier) @property)
