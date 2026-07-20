package goala

import (
	"fmt"
	"sync"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

var (
	languageOnce sync.Once
	language     *gotreesitter.Language
	languageErr  error
)

// Language returns the cached pure-Go *gotreesitter.Language compiled from the
// goala grammar. The first call builds the parse tables; subsequent calls are
// free. This is the in-process, CGO-free parser goala's own tooling uses.
func Language() (*gotreesitter.Language, error) {
	languageOnce.Do(func() {
		language, languageErr = GenerateLanguage(Grammar())
		if languageErr != nil {
			languageErr = fmt.Errorf("goala: build language: %w", languageErr)
		}
	})
	return language, languageErr
}

// MustLanguage is Language but panics on error. Convenient for callers that
// treat a broken grammar as a programming error (tests, CLI).
func MustLanguage() *gotreesitter.Language {
	lang, err := Language()
	if err != nil {
		panic(err)
	}
	return lang
}

// Parse parses goala source into a *gotreesitter.Tree using the cached language.
func Parse(source []byte) (*gotreesitter.Tree, error) {
	lang, err := Language()
	if err != nil {
		return nil, err
	}
	p := gotreesitter.NewParser(lang)
	return p.Parse(source)
}

// ParseSExpr parses source and returns the named-node S-expression of the root,
// the canonical form the C-parity harness byte-compares against.
func ParseSExpr(source []byte) (string, error) {
	lang, err := Language()
	if err != nil {
		return "", err
	}
	tree, err := Parse(source)
	if err != nil {
		return "", err
	}
	return tree.RootNode().SExpr(lang), nil
}
