package main

import (
	"fmt"
	"os"

	"bpl-plus/interpreter"
	"bpl-plus/lexer"
	"bpl-plus/parser"
)

// compileAndRun is used by the CLI path (fresh interpreter per file).
func compileAndRun(filename string, src string, sourceLines []string) error {
	// Use your interpreter's source-aware constructor so runtime errors show caret lines.
	in := interpreter.NewWithSource(filename, src)

	lx := lexer.New(src)
	ps := parser.New(lx)

	prog, err := ps.ParseProgram()
	if err != nil {
		// Parser errors are plain errors (not runtimeErr formatted), so print here.
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}

	if err := in.Run(prog); err != nil {
		// RuntimeError.Error() already renders nicely with caret + stack.
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}

	return nil
}

// compileAndRunWith is used by the REPL path (reuses one interpreter for session state).
func compileAndRunWith(session *interpreter.Interpreter, filename string, src string) error {
	// Update the interpreter's current source context so runtime errors show the right caret line.
	session.SetSource(filename, src)

	lx := lexer.New(src)
	ps := parser.New(lx)

	prog, err := ps.ParseProgram()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}

	if err := session.Run(prog); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return err
	}

	return nil
}
