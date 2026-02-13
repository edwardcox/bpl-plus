package main

import (
	"bpl-plus/interpreter"
	"bpl-plus/lexer"
	"bpl-plus/parser"
)

// compileAndRun is used for normal file execution (fresh interpreter each time).
func compileAndRun(filename, src string, sourceLines []string) error {
	_ = sourceLines // interpreter splits internally
	in := interpreter.NewWithSource(filename, src)
	return compileAndRunWith(in, filename, src)
}

// compileAndRunWith runs code using an existing interpreter instance.
// This is what makes the REPL stateful across inputs.
func compileAndRunWith(in *interpreter.Interpreter, filename, src string) error {
	lx := lexer.New(src)

	ps := parser.New(lx)
	stmts, err := ps.ParseProgram()
	if err != nil {
		return err
	}

	// Ensure runtime errors + imports have the right context for this chunk.
	in.SetSource(filename, src)

	return in.Run(stmts)
}
