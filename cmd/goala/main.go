// Command goala is the goala toolchain front door. In Phase 1 it exposes the
// `grammar` subcommand family (emit / parse / doctor) — thin wrappers over the
// grammargen APIs that project the single goala Grammar() onto every artifact.
// Transpiler subcommands (emit/run/build) arrive in Phase 2.
package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Fprint(os.Stderr, `goala — statically-typed functional language transpiling to Go

usage:
  goala grammar emit    [flags]   emit parser.c / blob / grammar.json / Go DSL / highlights
  goala grammar parse   [flags]   parse goala source and print the S-expression tree
  goala grammar doctor            report grammar conflicts and generation stats

run "goala grammar <sub> -h" for subcommand flags.
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "grammar":
		os.Exit(runGrammar(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "goala: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
