package main

import (
	"fmt"
	"os"

	"bpl-plus/interpreter"
	"bpl-plus/lexer"
	"bpl-plus/parser"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: bpl run <file.bpl>")
		os.Exit(1)
	}

	if os.Args[1] != "run" {
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}

	file := os.Args[2]

	srcBytes, err := os.ReadFile(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}
	src := string(srcBytes)

	lx := lexer.New(src)
	p := parser.New(lx)

	prog, err := p.ParseProgram()
	if err != nil {
		fmt.Println("Parse error:", err)
		os.Exit(1)
	}

	interp := interpreter.NewWithSource(file, src)
	if err := interp.Run(prog); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
