package goala

import "github.com/odvcencio/gotreesitter/grammargen"

// ImportRules copies the named root rules from src into dst, together with the
// transitive closure of every rule they reference via Sym(), deep-cloning each
// rule subtree so the two grammars never share mutable node pointers.
//
// It is the "compose grammars by subtree" primitive that ExtendGrammar lacks:
// ExtendGrammar copies an ENTIRE base grammar and lets you override pieces,
// which is right for building a superset (ferrous-wheel over Go). ImportRules
// is right for AUTHORING A NEW grammar that only BORROWS a few sub-grammars —
// goala borrows Go's lexical tokens (identifier, comment, numeric/string/rune
// literals) while authoring its own spine from NewGrammar up.
//
// Boundary rule: a referenced symbol is NOT imported (and its subtree is not
// walked) if dst already defines it. This lets the destination grammar own the
// meaning of shared nonterminals — e.g. goala defines its own `_expression`, so
// importing a Go rule that references `_expression` binds to goala's expression
// grammar instead of dragging Go's entire statement/expression closure along.
//
// Imported rules are appended to dst in dependency-stable order (roots first,
// then newly-discovered dependencies in encounter order). The start rule of dst
// (grammargen uses the first defined rule) is therefore never disturbed by an
// import, provided at least one rule was defined before ImportRules is called.
//
// Modeled on grammargen's ExtendGrammar deep-copy discipline (grammar.go:322,
// cloneRule loop). Proposed for upstreaming into grammargen next to
// ExtendGrammar as the missing subtree-composition primitive.
func ImportRules(dst, src *grammargen.Grammar, roots ...string) {
	seen := make(map[string]bool)
	var queue []string

	enqueue := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		// Never overwrite a rule the destination already owns, and do not walk
		// into the source's definition of it: the destination's meaning wins.
		if _, exists := dst.Rules[name]; exists {
			return
		}
		queue = append(queue, name)
	}

	for _, r := range roots {
		enqueue(r)
	}

	for i := 0; i < len(queue); i++ {
		name := queue[i]
		srcRule, ok := src.Rules[name]
		if !ok {
			// Root or dependency missing from source: skip silently. Callers can
			// assert presence via a test; a missing borrowed root usually means a
			// typo, which the conflict/parse tests will surface immediately.
			continue
		}
		cloned := cloneImportedRule(srcRule)
		dst.Define(name, cloned)
		for _, dep := range symbolRefs(srcRule) {
			enqueue(dep)
		}
	}
}

// symbolRefs returns every symbol name referenced (via Sym) anywhere in the
// rule subtree, in a stable pre-order.
func symbolRefs(r *grammargen.Rule) []string {
	var out []string
	var walk func(*grammargen.Rule)
	walk = func(n *grammargen.Rule) {
		if n == nil {
			return
		}
		if n.Kind == grammargen.RuleSymbol && n.Value != "" {
			out = append(out, n.Value)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(r)
	return out
}

// cloneImportedRule deep-copies a rule subtree, mirroring grammargen.cloneRule
// (which is unexported). Kept local so ImportRules can live in goala before the
// helper is upstreamed.
func cloneImportedRule(r *grammargen.Rule) *grammargen.Rule {
	if r == nil {
		return nil
	}
	cp := *r
	if len(r.Children) > 0 {
		cp.Children = make([]*grammargen.Rule, len(r.Children))
		for i, c := range r.Children {
			cp.Children[i] = cloneImportedRule(c)
		}
	}
	return &cp
}
