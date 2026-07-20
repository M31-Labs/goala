// Package register wires goala into the gotreesitter grammars registry. Blank-
// importing it (`import _ "m31labs.dev/goala/register"`) is sufficient for
// grammars.DetectLanguage to resolve ".goala" files to goala's language, which
// is the single hook canopy and every registry-driven tool use to index, search,
// graph, and serve code intelligence over goala source — with zero goala-specific
// code inside those tools.
//
// It lives in a subpackage so the goala core stays free of the grammars registry
// dependency: only consumers that want detection pay for it.
package register

import (
	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	goala "m31labs.dev/goala"
)

func init() {
	grammars.RegisterExtension(grammars.ExtensionEntry{
		Name:       "goala",
		Extensions: []string{".goala"},
		Aliases:    []string{"goala"},
		GenerateLanguage: func() (*gotreesitter.Language, error) {
			return goala.Language()
		},
		HighlightQuery: goala.HighlightQuery(),
	})
}
