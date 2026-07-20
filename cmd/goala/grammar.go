package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/odvcencio/gotreesitter/grammargen"
	goala "m31labs.dev/goala"
)

func runGrammar(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "goala grammar: expected a subcommand (emit|parse|doctor)")
		return 2
	}
	switch args[0] {
	case "emit":
		return grammarEmit(args[1:])
	case "parse":
		return grammarParse(args[1:])
	case "doctor":
		return grammarDoctor(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "goala grammar: unknown subcommand %q\n", args[0])
		return 2
	}
}

// grammarEmit projects Grammar() onto the five emit targets (DESIGN §3.3).
func grammarEmit(args []string) int {
	fs := flag.NewFlagSet("goala grammar emit", flag.ContinueOnError)
	out := fs.String("out", "dist", "base output directory used by -all")
	all := fs.Bool("all", false, "emit all artifacts under -out using the standard tree-sitter-goala layout")
	cPath := fs.String("c", "", "write parser.c (tree-sitter ABI 14/15) to this path")
	binPath := fs.String("bin", "", "write the gotreesitter grammar blob to this path")
	jsonPath := fs.String("json", "", "write the resolved grammar.json to this path")
	goPath := fs.String("go", "", "write the round-tripped Go DSL source to this path")
	hlPath := fs.String("highlight", "", "write the inferred highlights.scm to this path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	type target struct {
		path string
		gen  func() ([]byte, error)
	}
	var targets []target

	if *all {
		root := filepath.Join(*out, "tree-sitter-goala")
		targets = append(targets,
			target{filepath.Join(root, "src", "parser.c"), bytesFromString(goala.EmitParserC)},
			target{filepath.Join(*out, "goala.bin"), goala.EmitBlob},
			target{filepath.Join(*out, "grammar.json"), goala.EmitGrammarJSON},
			target{filepath.Join(*out, "grammar_generated.go"), goala.EmitGrammarGoSource},
			target{filepath.Join(root, "queries", "highlights.scm"), func() ([]byte, error) { return []byte(goala.HighlightQuery()), nil }},
		)
	}
	if *cPath != "" {
		targets = append(targets, target{*cPath, bytesFromString(goala.EmitParserC)})
	}
	if *binPath != "" {
		targets = append(targets, target{*binPath, goala.EmitBlob})
	}
	if *jsonPath != "" {
		targets = append(targets, target{*jsonPath, goala.EmitGrammarJSON})
	}
	if *goPath != "" {
		targets = append(targets, target{*goPath, goala.EmitGrammarGoSource})
	}
	if *hlPath != "" {
		targets = append(targets, target{*hlPath, func() ([]byte, error) { return []byte(goala.HighlightQuery()), nil }})
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "goala grammar emit: nothing to do — pass -all or one of -c/-bin/-json/-go/-highlight")
		return 2
	}

	for _, t := range targets {
		data, err := t.gen()
		if err != nil {
			fmt.Fprintf(os.Stderr, "goala grammar emit: %v\n", err)
			return 1
		}
		if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "goala grammar emit: mkdir %s: %v\n", filepath.Dir(t.path), err)
			return 1
		}
		if err := os.WriteFile(t.path, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "goala grammar emit: write %s: %v\n", t.path, err)
			return 1
		}
		fmt.Printf("wrote %s (%d bytes)\n", t.path, len(data))
	}
	return 0
}

func bytesFromString(f func() (string, error)) func() ([]byte, error) {
	return func() ([]byte, error) {
		s, err := f()
		if err != nil {
			return nil, err
		}
		return []byte(s), nil
	}
}

// grammarParse parses goala source (from -text, -file, or stdin) and prints the
// S-expression tree — the interactive grammar-feedback loop (DESIGN §3.2).
func grammarParse(args []string) int {
	fs := flag.NewFlagSet("goala grammar parse", flag.ContinueOnError)
	text := fs.String("text", "", "goala source to parse")
	file := fs.String("file", "", "path to a .goala file to parse")
	format := fs.String("format", "sexpr", "output format: sexpr")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var src []byte
	var err error
	switch {
	case *text != "":
		src = []byte(*text)
	case *file != "":
		src, err = os.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goala grammar parse: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintln(os.Stderr, "goala grammar parse: pass -text or -file")
		return 2
	}

	if *format != "sexpr" {
		fmt.Fprintf(os.Stderr, "goala grammar parse: unknown -format %q (only sexpr)\n", *format)
		return 2
	}
	pretty, err := goala.PrettyParse(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goala grammar parse: %v\n", err)
		return 1
	}
	fmt.Println(pretty)
	return 0
}

// grammarDoctor reports conflict/generation stats — the diagnostics showcase.
func grammarDoctor(args []string) int {
	rep, err := grammargen.GenerateWithReport(goala.Grammar())
	if err != nil {
		fmt.Fprintf(os.Stderr, "goala grammar doctor: %v\n", err)
		return 1
	}
	glr := 0
	for _, c := range rep.Conflicts {
		if c.Resolution == "GLR (multiple actions kept)" {
			glr++
		}
	}
	fmt.Printf("goala grammar: symbols=%d states=%d tokens=%d\n", rep.SymbolCount, rep.StateCount, rep.TokenCount)
	fmt.Printf("conflicts: %d total, %d kept for GLR, %d resolved deterministically\n",
		len(rep.Conflicts), glr, len(rep.Conflicts)-glr)
	if len(rep.Warnings) > 0 {
		fmt.Printf("warnings: %d\n", len(rep.Warnings))
		for _, w := range rep.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	if glr != 0 {
		fmt.Fprintln(os.Stderr, "grammar is not LR-deterministic: some conflicts require the GLR runtime")
		return 1
	}
	fmt.Println("grammar is fully LR-deterministic (0 conflicts kept for GLR)")
	return 0
}
