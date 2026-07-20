package goala

import "strings"

// PrettySExpr reformats a single-line tree-sitter S-expression (as produced by
// gotreesitter's Node.SExpr) into an indented, one-node-per-line form. It is
// purely cosmetic — the token stream is identical to the input — but makes the
// golden corpus snapshots reviewable and their diffs legible.
//
// Example:
//
//	(source_file (function_declaration (identifier) (block)))
//
// becomes
//
//	(source_file
//	  (function_declaration
//	    (identifier)
//	    (block)))
func PrettySExpr(s string) string {
	var b strings.Builder
	depth := 0
	for i := 0; i < len(s); {
		switch c := s[i]; c {
		case '(':
			if b.Len() > 0 {
				b.WriteByte('\n')
				for k := 0; k < depth; k++ {
					b.WriteString("  ")
				}
			}
			b.WriteByte('(')
			depth++
			// Emit the node type name that immediately follows '('.
			i++
			for i < len(s) && s[i] != ' ' && s[i] != '(' && s[i] != ')' {
				b.WriteByte(s[i])
				i++
			}
		case ')':
			b.WriteByte(')')
			depth--
			i++
		case ' ':
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// PrettyParse parses goala source and returns the indented S-expression of the
// root node — the canonical golden-snapshot form used by the corpus tests and
// the `goala grammar parse -format sexpr` CLI.
func PrettyParse(source []byte) (string, error) {
	sx, err := ParseSExpr(source)
	if err != nil {
		return "", err
	}
	return PrettySExpr(sx), nil
}
