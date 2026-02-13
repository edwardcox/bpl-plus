package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"bpl-plus/interpreter"
	"github.com/chzyer/readline"
)

func runREPL() error {
	home, _ := os.UserHomeDir()
	histPath := ""
	if home != "" {
		histPath = filepath.Join(home, ".bplplus_history")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "bpl> ",
		HistoryFile:            histPath,
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
		HistorySearchFold:      true,
		DisableAutoSaveHistory: false,
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	fmt.Println("BPL+ REPL — :help for commands, :quit to exit.")
	fmt.Println("Arrow keys + history enabled. Multi-line blocks supported (if/while/for/function ... end).")
	fmt.Println("Paste Mode: type :paste, then end with '.' or :endpaste")
	fmt.Println("Introspection: :vars  :funcs  :modules")
	fmt.Println()

	// ✅ Single interpreter for the whole REPL session (stateful)
	session := interpreter.New()

	var buf strings.Builder
	depth := 0
	chunk := 0

	// Paste mode state
	pasteMode := false
	var pasteBuf strings.Builder

	for {
		if pasteMode {
			rl.SetPrompt(pastePrompt())
		} else {
			rl.SetPrompt(replPrompt(depth))
		}

		line, err := rl.Readline()

		// Ctrl+C
		if err == readline.ErrInterrupt {
			if pasteMode {
				pasteMode = false
				pasteBuf.Reset()
				fmt.Println("^C (paste cancelled)")
				continue
			}
			if buf.Len() > 0 || depth > 0 {
				buf.Reset()
				depth = 0
				fmt.Println("^C (buffer cleared)")
			}
			continue
		}

		// Ctrl+D
		if err == io.EOF {
			fmt.Println()
			return nil
		}
		if err != nil {
			return err
		}

		trim := strings.TrimSpace(line)

		// ---- PASTE MODE ----
		if pasteMode {
			// End paste session
			if trim == "." || trim == ":endpaste" {
				src := pasteBuf.String()
				pasteBuf.Reset()
				pasteMode = false

				if strings.TrimSpace(src) == "" {
					fmt.Println("(paste buffer empty)")
					continue
				}

				chunk++
				filename := replChunkFilename(chunk)
				_ = compileAndRunWith(session, filename, src)
				continue
			}

			// Cancel paste session without running
			if trim == ":cancel" {
				pasteBuf.Reset()
				pasteMode = false
				fmt.Println("(paste cancelled)")
				continue
			}

			// Accumulate raw lines
			pasteBuf.WriteString(line)
			pasteBuf.WriteString("\n")
			continue
		}

		// ---- NORMAL MODE ----

		// Commands only when not buffering a block.
		if depth == 0 && buf.Len() == 0 && strings.HasPrefix(trim, ":") {
			handled, cmdErr := handleREPLCommand(trim, &buf, &depth, &pasteMode, &pasteBuf, session)
			if handled {
				if cmdErr != nil {
					fmt.Fprintln(os.Stderr, cmdErr.Error())
				}
				continue
			}
		}

		// Accumulate input
		buf.WriteString(line)
		buf.WriteString("\n")

		// Update depth heuristic for multi-line blocks.
		depth = updateDepth(depth, trim)

		if depth > 0 {
			continue
		}

		src := buf.String()
		if strings.TrimSpace(src) == "" {
			buf.Reset()
			continue
		}
		buf.Reset()

		chunk++
		filename := replChunkFilename(chunk)
		_ = compileAndRunWith(session, filename, src)
	}
}

func replChunkFilename(chunk int) string {
	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "."
	}
	return filepath.Join(cwd, fmt.Sprintf("<repl:%d>", chunk))
}

func replPrompt(depth int) string {
	cwd, err := os.Getwd()
	base := "bpl"
	if err == nil && cwd != "" {
		base = filepath.Base(cwd)
	}

	if depth > 0 {
		return fmt.Sprintf("...[%s]> ", base)
	}
	return fmt.Sprintf("bpl[%s]> ", base)
}

func pastePrompt() string {
	cwd, err := os.Getwd()
	base := "bpl"
	if err == nil && cwd != "" {
		base = filepath.Base(cwd)
	}
	return fmt.Sprintf("paste[%s]> ", base)
}

func handleREPLCommand(
	cmd string,
	buf *strings.Builder,
	depth *int,
	pasteMode *bool,
	pasteBuf *strings.Builder,
	session *interpreter.Interpreter,
) (bool, error) {
	switch {
	case cmd == ":q" || cmd == ":quit" || cmd == ":exit":
		os.Exit(0)
		return true, nil

	case cmd == ":h" || cmd == ":help":
		fmt.Println("Commands:")
		fmt.Println("  :help              Show this help")
		fmt.Println("  :quit              Exit the REPL")
		fmt.Println("  :pwd               Print current directory")
		fmt.Println("  :cd <dir>           Change directory")
		fmt.Println("  :load <file>        Run a .bpl file (fresh interpreter, like CLI)")
		fmt.Println("  :reset              Clear buffered multi-line input")
		fmt.Println("  :clear              Clear the screen")
		fmt.Println("  :paste              Start paste mode (end with '.' or :endpaste)")
		fmt.Println("  :vars               Show global variables (REPL session)")
		fmt.Println("  :funcs              Show user-defined functions (REPL session)")
		fmt.Println("  :modules            Show module load state (REPL session)")
		fmt.Println()
		fmt.Println("Paste mode controls:")
		fmt.Println("  .                   End + run pasted program")
		fmt.Println("  :endpaste           End + run pasted program")
		fmt.Println("  :cancel             Cancel paste without running")
		fmt.Println()
		fmt.Println("Notes:")
		fmt.Println("  - Multi-line blocks: if/while/for/function ... end")
		fmt.Println("  - Relative imports resolve from your current directory.")
		fmt.Println("  - REPL input shares state across runs (vars/functions/modules persist).")
		return true, nil

	case cmd == ":pwd":
		cwd, err := os.Getwd()
		if err != nil {
			return true, err
		}
		fmt.Println(cwd)
		return true, nil

	case strings.HasPrefix(cmd, ":cd "):
		dir := strings.TrimSpace(strings.TrimPrefix(cmd, ":cd "))
		if dir == "" {
			return true, fmt.Errorf("Usage: :cd <dir>")
		}
		if err := os.Chdir(dir); err != nil {
			return true, err
		}
		return true, nil

	case strings.HasPrefix(cmd, ":load "):
		path := strings.TrimSpace(strings.TrimPrefix(cmd, ":load "))
		if path == "" {
			return true, fmt.Errorf("Usage: :load <file.bpl>")
		}

		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return true, fmt.Errorf("Failed to read %s: %s", path, err.Error())
		}

		// Make imports resolve relative to the loaded file directory.
		if dir := filepath.Dir(path); dir != "" {
			_ = os.Chdir(dir)
		}

		// Load runs like the CLI: fresh interpreter for the file
		return true, runSource(path, string(b))

	case cmd == ":reset":
		buf.Reset()
		*depth = 0
		fmt.Println("(buffer cleared)")
		return true, nil

	case cmd == ":clear":
		fmt.Print("\033[2J\033[H")
		return true, nil

	case cmd == ":paste":
		buf.Reset()
		*depth = 0
		pasteBuf.Reset()
		*pasteMode = true
		fmt.Println("(paste mode: end with '.' or :endpaste, cancel with :cancel)")
		return true, nil

	case cmd == ":vars":
		globs := session.GlobalsSnapshot()
		if len(globs) == 0 {
			fmt.Println("(no globals)")
			return true, nil
		}
		keys := make([]string, 0, len(globs))
		for k := range globs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%s = %v\n", k, globs[k])
		}
		return true, nil

	case cmd == ":funcs":
		names := session.FuncNames()
		if len(names) == 0 {
			fmt.Println("(no user functions)")
			return true, nil
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return true, nil

	case cmd == ":modules":
		loading, loaded := session.ModulesSnapshot()
		if len(loading) == 0 && len(loaded) == 0 {
			fmt.Println("(no modules loaded)")
			return true, nil
		}
		if len(loading) > 0 {
			fmt.Println("loading:")
			for _, p := range loading {
				fmt.Println("  " + p)
			}
		}
		if len(loaded) > 0 {
			fmt.Println("loaded:")
			for _, p := range loaded {
				fmt.Println("  " + p)
			}
		}
		return true, nil

	default:
		fmt.Println("Unknown command. Try :help")
		return true, nil
	}
}

func updateDepth(depth int, trimmed string) int {
	if trimmed == "" {
		return depth
	}

	low := strings.ToLower(trimmed)

	if strings.HasPrefix(low, "#") {
		return depth
	}

	if isBlockOpener(low) {
		return depth + 1
	}

	if low == "end" {
		if depth > 0 {
			return depth - 1
		}
		return 0
	}

	if low == "else" {
		return depth
	}

	return depth
}

func isBlockOpener(low string) bool {
	return strings.HasPrefix(low, "if ") ||
		strings.HasPrefix(low, "while ") ||
		strings.HasPrefix(low, "for ") ||
		strings.HasPrefix(low, "function ")
}
