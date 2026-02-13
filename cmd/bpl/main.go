package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]

	// REPL mode: no args
	if len(args) == 0 {
		if err := runREPL(); err != nil {
			// REPL errors are usually already printed nicely, but keep a fallback.
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		return
	}

	// Compatibility: `bplplus run file.bpl`
	if len(args) >= 2 && args[0] == "run" {
		args = args[1:]
	}

	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  bplplus <file.bpl>")
		fmt.Fprintln(os.Stderr, "  bplplus run <file.bpl>")
		fmt.Fprintln(os.Stderr, "  bplplus           # REPL")
		os.Exit(2)
	}

	filename := args[0]
	abs, err := filepath.Abs(filename)
	if err == nil {
		filename = abs
	}

	srcBytes, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %s\n", err.Error())
		os.Exit(1)
	}

	src := string(srcBytes)

	// Important: Set CWD to file directory so imports resolve intuitively.
	// (If your import resolver is already based on importer filename, this is still a nice extra.)
	if dir := filepath.Dir(filename); dir != "" {
		_ = os.Chdir(dir)
	}

	if err := runSource(filename, src); err != nil {
		// runSource should already print formatted errors if your interpreter does that.
		// But we still exit non-zero.
		os.Exit(1)
	}
}

// runSource is the ONLY place you should need to adapt names if your package APIs differ.
// Keep everything else stable.
func runSource(filename, src string) error {
	// If your interpreter expects source lines for caret rendering,
	// keep this split exactly as-is.
	sourceLines := splitLinesKeepIndex(src)

	// --- ADAPTER POINTS START ---
	// Replace these calls with your actual constructors/methods if they differ.

	// Example (common pattern):
	// lx := lexer.New(filename, src)
	// ps := parser.New(lx)
	// program, err := ps.ParseProgram()
	// if err != nil { return err }
	// in := interpreter.New(filename, sourceLines)
	// return in.Exec(program)

	// For now we call a helper that you will wire to your actual pipeline.
	return compileAndRun(filename, src, sourceLines)

	// --- ADAPTER POINTS END ---
}

// splitLinesKeepIndex makes 1-based line access easy in error renderers.
// It returns a slice where index i corresponds to line i (1-based). Index 0 is unused.
func splitLinesKeepIndex(src string) []string {
	raw := strings.Split(src, "\n")
	lines := make([]string, 1, len(raw)+1)
	lines = append(lines, raw...)
	return lines
}
