package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bpl-plus/interpreter"
	"bpl-plus/lexer"
	"bpl-plus/parser"
)

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  bplplus <file.bpl>")
	fmt.Println("  bplplus run <file.bpl>")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	args := os.Args[1:]

	var filename string

	// Allow: bplplus run file.bpl
	if args[0] == "run" {
		if len(args) != 2 {
			usage()
		}
		filename = args[1]
	} else {
		// Allow: bplplus file.bpl
		filename = args[0]
	}

	if !strings.HasSuffix(filename, ".bpl") {
		fmt.Fprintf(os.Stderr, "Error: expected a .bpl file, got %q\n", filename)
		os.Exit(1)
	}

	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filename, err)
		os.Exit(1)
	}

	lx := lexer.New(string(src))
	p := parser.New(lx)

	stmts, err := p.ParseProgram()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	interp := interpreter.NewWithSource(
		filepath.Base(filename),
		string(src),
	)

	if err := interp.Run(stmts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
