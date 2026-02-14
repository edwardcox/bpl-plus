// interpreter/interpreter.go
package interpreter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"bpl-plus/ast"
	"bpl-plus/lexer"
	"bpl-plus/parser"
)

type ValueKind int

const (
	ValNull ValueKind = iota
	ValNumber
	ValString
	ValBool
	ValArray
	ValMap
)

// ArrayObject gives arrays reference semantics.
type ArrayObject struct {
	Elems []Value
}

// MapObject gives maps reference semantics.
type MapObject struct {
	Elems map[string]Value
}

type Value struct {
	Kind   ValueKind
	Number float64
	Str    string
	Bool   bool
	Arr    *ArrayObject
	Map    *MapObject
}

func NullValue() Value            { return Value{Kind: ValNull} }
func NumberValue(n float64) Value { return Value{Kind: ValNumber, Number: n} }
func StringValue(s string) Value  { return Value{Kind: ValString, Str: s} }
func BoolValue(b bool) Value      { return Value{Kind: ValBool, Bool: b} }
func ArrayValue(elems []Value) Value {
	return Value{Kind: ValArray, Arr: &ArrayObject{Elems: elems}}
}
func MapValue(m map[string]Value) Value {
	if m == nil {
		m = map[string]Value{}
	}
	return Value{Kind: ValMap, Map: &MapObject{Elems: m}}
}

func (v Value) arrayElems() []Value {
	if v.Kind != ValArray || v.Arr == nil {
		return nil
	}
	return v.Arr.Elems
}

func (v Value) mapElems() map[string]Value {
	if v.Kind != ValMap || v.Map == nil {
		return nil
	}
	return v.Map.Elems
}

func (v Value) ToString() string {
	switch v.Kind {
	case ValNumber:
		if v.Number == float64(int64(v.Number)) {
			return fmt.Sprintf("%d", int64(v.Number))
		}
		return fmt.Sprintf("%g", v.Number)

	case ValString:
		return v.Str

	case ValBool:
		if v.Bool {
			return "true"
		}
		return "false"

	case ValArray:
		elems := v.arrayElems()
		var b strings.Builder
		b.WriteString("[")
		for idx, el := range elems {
			if idx > 0 {
				b.WriteString(", ")
			}
			b.WriteString(el.ToString())
		}
		b.WriteString("]")
		return b.String()

	case ValMap:
		m := v.mapElems()
		if m == nil {
			return "{}"
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var b strings.Builder
		b.WriteString("{")
		for idx, k := range keys {
			if idx > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%q: %s", k, m[k].ToString()))
		}
		b.WriteString("}")
		return b.String()

	default:
		return "null"
	}
}

type ReturnSignal struct{ Val Value }

func (r ReturnSignal) Error() string { return "return" }

type BreakSignal struct{}

func (b BreakSignal) Error() string { return "break" }

type ContinueSignal struct{}

func (c ContinueSignal) Error() string { return "continue" }

type RuntimeError struct {
	File  string
	Span  ast.Span
	Msg   string
	Line  string
	Stack []string
}

func (e RuntimeError) Error() string {
	loc := "unknown:0:0"
	if e.File != "" && e.Span.Line > 0 && e.Span.Col > 0 {
		loc = fmt.Sprintf("%s:%d:%d", e.File, e.Span.Line, e.Span.Col)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Runtime error at %s\n", loc))
	b.WriteString(fmt.Sprintf("  %s\n", e.Msg))

	if e.Line != "" && e.Span.Line > 0 {
		b.WriteString(fmt.Sprintf("  %d | %s\n", e.Span.Line, e.Line))

		prefix := fmt.Sprintf("  %d | ", e.Span.Line)
		caretSpaces := len(prefix) + (e.Span.Col - 1)
		if caretSpaces < 0 {
			caretSpaces = 0
		}
		b.WriteString(strings.Repeat(" ", caretSpaces))
		b.WriteString("^\n")
	}

	if len(e.Stack) > 0 {
		b.WriteString("Stack:\n")
		for _, fn := range e.Stack {
			b.WriteString(fmt.Sprintf("  at %s()\n", fn))
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

type moduleState int

const (
	modNone moduleState = iota
	modLoading
	modLoaded
)

type Interpreter struct {
	globals map[string]Value
	locals  []map[string]Value
	funcs   map[string]*ast.FunctionDecl

	in *bufio.Reader

	filename string
	lines    []string

	callStack []string

	modules     map[string]moduleState
	moduleStack []string

	// File handles: #n -> *os.File
	files map[int]*os.File
	// Buffered readers for handles (created on demand)
	readers map[int]*bufio.Reader
}

func NewWithSource(filename string, source string) *Interpreter {
	return &Interpreter{
		globals:     map[string]Value{},
		locals:      []map[string]Value{},
		funcs:       map[string]*ast.FunctionDecl{},
		in:          bufio.NewReader(os.Stdin),
		filename:    filename,
		lines:       splitLinesPreserve(source),
		callStack:   []string{},
		modules:     map[string]moduleState{},
		moduleStack: []string{},
		files:       map[int]*os.File{},
		readers:     map[int]*bufio.Reader{},
	}
}

func New() *Interpreter { return NewWithSource("", "") }

func splitLinesPreserve(src string) []string {
	if src == "" {
		return []string{}
	}
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	return strings.Split(src, "\n")
}

func (i *Interpreter) inFunction() bool { return len(i.locals) > 0 }

func (i *Interpreter) currentEnv() map[string]Value {
	if i.inFunction() {
		return i.locals[len(i.locals)-1]
	}
	return i.globals
}

func (i *Interpreter) pushLocals() { i.locals = append(i.locals, map[string]Value{}) }
func (i *Interpreter) popLocals()  { i.locals = i.locals[:len(i.locals)-1] }

func (i *Interpreter) Run(stmts []ast.Stmt) error {
	for _, s := range stmts {
		if err := i.execStmt(s); err != nil {
			switch err.(type) {
			case ReturnSignal, BreakSignal, ContinueSignal:
				return err
			default:
				return err
			}
		}
	}
	return nil
}

func (i *Interpreter) runtimeErr(span ast.Span, msg string) error {
	lineText := ""
	if span.Line > 0 && span.Line-1 < len(i.lines) {
		lineText = i.lines[span.Line-1]
	}

	stack := make([]string, 0, len(i.callStack))
	for idx := len(i.callStack) - 1; idx >= 0; idx-- {
		stack = append(stack, i.callStack[idx])
	}

	return RuntimeError{
		File:  i.filename,
		Span:  span,
		Msg:   msg,
		Line:  lineText,
		Stack: stack,
	}
}

// Find the environment a variable lives in (locals first, then globals).
func (i *Interpreter) findVarEnv(name string) (map[string]Value, Value, bool) {
	if i.inFunction() {
		env := i.currentEnv()
		if v, ok := env[name]; ok {
			return env, v, true
		}
	}
	if v, ok := i.globals[name]; ok {
		return i.globals, v, true
	}
	return nil, Value{}, false
}

func (i *Interpreter) execStmt(s ast.Stmt) error {
	switch stmt := s.(type) {
	case *ast.ImportStmt:
		return i.execImport(stmt)

	case *ast.FunctionDecl:
		i.funcs[stmt.Name] = stmt
		return nil

	case *ast.ReturnStmt:
		if !i.inFunction() {
			return i.runtimeErr(stmt.GetSpan(), "Return is only valid inside a function")
		}
		val, err := i.evalExpr(stmt.Value)
		if err != nil {
			return err
		}
		return ReturnSignal{Val: val}

	case *ast.BreakStmt:
		return BreakSignal{}

	case *ast.ContinueStmt:
		return ContinueSignal{}

	case *ast.AssignStmt:
		val, err := i.evalExpr(stmt.Value)
		if err != nil {
			return err
		}
		i.currentEnv()[stmt.Name] = val
		return nil

	case *ast.IndexAssignStmt:
		return i.execIndexAssign(stmt)

	case *ast.ExprStmt:
		_, err := i.evalExpr(stmt.Expr)
		return err

	case *ast.PrintStmt:
		val, err := i.evalExpr(stmt.Value)
		if err != nil {
			return err
		}
		fmt.Println(val.ToString())
		return nil

	case *ast.OpenStmt:
		return i.execOpen(stmt)

	case *ast.CloseStmt:
		return i.execClose(stmt)

	case *ast.PrintHandleStmt:
		return i.execPrintHandle(stmt)

	case *ast.IfStmt:
		cond, err := i.evalExpr(stmt.Condition)
		if err != nil {
			return err
		}
		if cond.Kind != ValBool {
			return i.runtimeErr(stmt.Condition.GetSpan(), "If condition must be boolean")
		}
		if cond.Bool {
			return i.Run(stmt.Then)
		}
		return i.Run(stmt.Else)

	case *ast.WhileStmt:
		for {
			cond, err := i.evalExpr(stmt.Condition)
			if err != nil {
				return err
			}
			if cond.Kind != ValBool {
				return i.runtimeErr(stmt.Condition.GetSpan(), "While condition must be boolean")
			}
			if !cond.Bool {
				break
			}
			err = i.Run(stmt.Body)
			if err != nil {
				switch err.(type) {
				case BreakSignal:
					return nil
				case ContinueSignal:
					continue
				default:
					return err
				}
			}
		}
		return nil

	case *ast.ForStmt:
		return i.execFor(stmt)

	case *ast.ForEachStmt:
		return i.execForEach(stmt)

	default:
		span, ok := ast.SpanOf(s)
		if !ok {
			span = ast.Span{}
		}
		return i.runtimeErr(span, fmt.Sprintf("Unsupported statement %s", s.NodeKind()))
	}
}

// ---------- File Handles ----------

func (i *Interpreter) execOpen(stmt *ast.OpenStmt) error {
	if stmt.Handle <= 0 {
		return i.runtimeErr(stmt.GetSpan(), "open handle must be a positive integer")
	}

	pathV, err := i.evalExpr(stmt.Path)
	if err != nil {
		return err
	}
	modeV, err := i.evalExpr(stmt.Mode)
	if err != nil {
		return err
	}
	if pathV.Kind != ValString {
		return i.runtimeErr(stmt.Path.GetSpan(), "open path must be a string")
	}
	if modeV.Kind != ValString {
		return i.runtimeErr(stmt.Mode.GetSpan(), "open mode must be a string (\"r\", \"w\", or \"a\")")
	}

	mode := strings.ToLower(strings.TrimSpace(modeV.Str))
	path := pathV.Str

	// if already open, close first
	if f, ok := i.files[stmt.Handle]; ok && f != nil {
		_ = f.Close()
	}
	delete(i.files, stmt.Handle)
	delete(i.readers, stmt.Handle)

	var f *os.File

	switch mode {
	case "w":
		// ✅ auto-create parent dirs
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		ff, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if e != nil {
			return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("open failed: %v", e))
		}
		f = ff

	case "a":
		// ✅ auto-create parent dirs
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		ff, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if e != nil {
			return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("open failed: %v", e))
		}
		f = ff

	case "r":
		ff, e := os.Open(path)
		if e != nil {
			return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("open failed: %v", e))
		}
		f = ff

	default:
		return i.runtimeErr(stmt.GetSpan(), "open mode must be \"r\", \"w\", or \"a\"")
	}

	i.files[stmt.Handle] = f
	// Reader will be created lazily (or immediately for read mode if you prefer).
	return nil
}

func (i *Interpreter) execClose(stmt *ast.CloseStmt) error {
	f, ok := i.files[stmt.Handle]
	if !ok || f == nil {
		return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("close failed: handle #%d is not open", stmt.Handle))
	}
	_ = f.Close()
	delete(i.files, stmt.Handle)
	delete(i.readers, stmt.Handle)
	return nil
}

func (i *Interpreter) execPrintHandle(stmt *ast.PrintHandleStmt) error {
	f, ok := i.files[stmt.Handle]
	if !ok || f == nil {
		return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("print failed: handle #%d is not open", stmt.Handle))
	}
	v, err := i.evalExpr(stmt.Value)
	if err != nil {
		return err
	}
	_, werr := f.WriteString(v.ToString() + "\n")
	if werr != nil {
		return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("print failed: %v", werr))
	}
	return nil
}

// ---------- Imports ----------

func (i *Interpreter) fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (i *Interpreter) projectRootCandidates() []string {
	return []string{"."}
}

func (i *Interpreter) importCandidates(raw string, importerFilename string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	withExt := raw
	needsExt := filepath.Ext(raw) == ""
	if needsExt {
		withExt = raw + ".bpl"
	}

	if filepath.IsAbs(raw) {
		cands := []string{filepath.Clean(raw)}
		if needsExt {
			cands = append(cands, filepath.Clean(withExt))
		}
		return cands
	}

	baseDir := ""
	if importerFilename != "" {
		baseDir = filepath.Dir(importerFilename)
	}

	cands := []string{}

	if baseDir != "" {
		cands = append(cands, filepath.Clean(filepath.Join(baseDir, raw)))
		if needsExt {
			cands = append(cands, filepath.Clean(filepath.Join(baseDir, withExt)))
		}

		cands = append(cands, filepath.Clean(filepath.Join(baseDir, "lib", raw)))
		if needsExt {
			cands = append(cands, filepath.Clean(filepath.Join(baseDir, "lib", withExt)))
		}
	}

	for _, root := range i.projectRootCandidates() {
		cands = append(cands, filepath.Clean(filepath.Join(root, raw)))
		if needsExt {
			cands = append(cands, filepath.Clean(filepath.Join(root, withExt)))
		}

		cands = append(cands, filepath.Clean(filepath.Join(root, "lib", raw)))
		if needsExt {
			cands = append(cands, filepath.Clean(filepath.Join(root, "lib", withExt)))
		}
	}

	seen := map[string]bool{}
	out := []string{}
	for _, c := range cands {
		if c == "" {
			continue
		}
		if seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func (i *Interpreter) resolveImportPath(raw string, importerFilename string) (string, []string) {
	cands := i.importCandidates(raw, importerFilename)
	for _, c := range cands {
		if i.fileExists(c) {
			return c, cands
		}
	}
	if len(cands) > 0 {
		return cands[0], cands
	}
	return raw, cands
}

func (i *Interpreter) circularImportMessage(target string) string {
	var b strings.Builder
	b.WriteString("Circular import detected:\n")
	for _, p := range i.moduleStack {
		b.WriteString("  ")
		b.WriteString(p)
		b.WriteString("\n")
	}
	b.WriteString("  ")
	b.WriteString(target)
	return strings.TrimRight(b.String(), "\n")
}

func (i *Interpreter) execImport(stmt *ast.ImportStmt) error {
	importerFile := i.filename
	resolved, tried := i.resolveImportPath(stmt.Path, importerFile)

	switch i.modules[resolved] {
	case modLoaded:
		return nil
	case modLoading:
		return i.runtimeErr(stmt.GetSpan(), i.circularImportMessage(resolved))
	}

	if !i.fileExists(resolved) {
		msg := fmt.Sprintf("import failed: file not found %q", stmt.Path)
		if len(tried) > 0 {
			msg += "\nTried:\n"
			for _, c := range tried {
				msg += "  " + c + "\n"
			}
			msg = strings.TrimRight(msg, "\n")
		}
		return i.runtimeErr(stmt.GetSpan(), msg)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("import failed for %q: %v", resolved, err))
	}

	lx := lexer.New(string(data))
	p := parser.New(lx)
	prog, err := p.ParseProgram()
	if err != nil {
		return err
	}

	i.modules[resolved] = modLoading
	i.moduleStack = append(i.moduleStack, resolved)

	prevFile := i.filename
	prevLines := i.lines

	i.filename = resolved
	i.lines = splitLinesPreserve(string(data))

	runErr := i.Run(prog)

	i.filename = prevFile
	i.lines = prevLines

	i.moduleStack = i.moduleStack[:len(i.moduleStack)-1]

	if runErr != nil {
		i.modules[resolved] = modNone
		return runErr
	}

	i.modules[resolved] = modLoaded
	return nil
}

// ---------- Arrays / Maps / Loops ----------

func (i *Interpreter) execIndexAssign(stmt *ast.IndexAssignStmt) error {
	env, containerVal, ok := i.findVarEnv(stmt.Name)
	if !ok {
		return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("Undefined variable %q", stmt.Name))
	}

	iv, err := i.evalExpr(stmt.Index)
	if err != nil {
		return err
	}

	newVal, err := i.evalExpr(stmt.Value)
	if err != nil {
		return err
	}

	if containerVal.Kind == ValArray && containerVal.Arr != nil {
		idx, err := i.toIndex(iv, stmt.Index.GetSpan())
		if err != nil {
			return err
		}

		elems := containerVal.Arr.Elems
		if idx < 0 || idx >= len(elems) {
			return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("Array index out of bounds (index %d, size %d)", idx, len(elems)))
		}

		containerVal.Arr.Elems[idx] = newVal
		env[stmt.Name] = containerVal
		return nil
	}

	if containerVal.Kind == ValMap && containerVal.Map != nil {
		if iv.Kind != ValString {
			return i.runtimeErr(stmt.Index.GetSpan(), "Map key must be a string")
		}
		if containerVal.Map.Elems == nil {
			containerVal.Map.Elems = map[string]Value{}
		}
		containerVal.Map.Elems[iv.Str] = newVal
		env[stmt.Name] = containerVal
		return nil
	}

	return i.runtimeErr(stmt.GetSpan(), "Index assignment requires an array or map")
}

func (i *Interpreter) execFor(stmt *ast.ForStmt) error {
	startV, err := i.evalExpr(stmt.Start)
	if err != nil {
		return err
	}
	endV, err := i.evalExpr(stmt.End)
	if err != nil {
		return err
	}
	if startV.Kind != ValNumber || endV.Kind != ValNumber {
		return i.runtimeErr(stmt.GetSpan(), "For loop start/end must be numbers")
	}

	step := 1.0
	if stmt.Step != nil {
		stepV, err := i.evalExpr(stmt.Step)
		if err != nil {
			return err
		}
		if stepV.Kind != ValNumber || stepV.Number == 0 {
			return i.runtimeErr(stmt.Step.GetSpan(), "For loop step must be a non-zero number")
		}
		step = stepV.Number
	} else if startV.Number > endV.Number {
		step = -1
	}

	i.currentEnv()[stmt.Var] = NumberValue(startV.Number)

	for {
		curV := i.currentEnv()[stmt.Var]
		if curV.Kind != ValNumber {
			return i.runtimeErr(stmt.GetSpan(), "For loop variable must remain numeric")
		}
		cur := curV.Number

		if (step > 0 && cur > endV.Number) || (step < 0 && cur < endV.Number) {
			break
		}

		err := i.Run(stmt.Body)
		if err != nil {
			switch err.(type) {
			case BreakSignal:
				return nil
			case ContinueSignal:
				i.currentEnv()[stmt.Var] = NumberValue(cur + step)
				continue
			default:
				return err
			}
		}

		i.currentEnv()[stmt.Var] = NumberValue(cur + step)
	}

	return nil
}

func (i *Interpreter) execForEach(stmt *ast.ForEachStmt) error {
	iterV, err := i.evalExpr(stmt.Iterable)
	if err != nil {
		return err
	}

	if iterV.Kind == ValArray && iterV.Arr != nil {
		for idx, el := range iterV.Arr.Elems {
			i.currentEnv()[stmt.Var] = el
			if stmt.IndexVar != "" {
				i.currentEnv()[stmt.IndexVar] = NumberValue(float64(idx))
			}
			err := i.Run(stmt.Body)
			if err != nil {
				switch err.(type) {
				case BreakSignal:
					return nil
				case ContinueSignal:
					continue
				default:
					return err
				}
			}
		}
		return nil
	}

	if iterV.Kind == ValMap && iterV.Map != nil {
		m := iterV.Map.Elems
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for idx, k := range keys {
			i.currentEnv()[stmt.Var] = StringValue(k)
			if stmt.IndexVar != "" {
				i.currentEnv()[stmt.IndexVar] = NumberValue(float64(idx))
			}
			err := i.Run(stmt.Body)
			if err != nil {
				switch err.(type) {
				case BreakSignal:
					return nil
				case ContinueSignal:
					continue
				default:
					return err
				}
			}
		}
		return nil
	}

	return i.runtimeErr(stmt.GetSpan(), "foreach expects an array or map")
}

func (i *Interpreter) valuesEqual(a, b Value) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case ValNull:
		return true
	case ValNumber:
		return a.Number == b.Number
	case ValString:
		return a.Str == b.Str
	case ValBool:
		return a.Bool == b.Bool
	case ValArray:
		if a.Arr == nil || b.Arr == nil {
			return a.Arr == b.Arr
		}
		if len(a.Arr.Elems) != len(b.Arr.Elems) {
			return false
		}
		for idx := range a.Arr.Elems {
			if !i.valuesEqual(a.Arr.Elems[idx], b.Arr.Elems[idx]) {
				return false
			}
		}
		return true
	case ValMap:
		if a.Map == nil || b.Map == nil {
			return a.Map == b.Map
		}
		am := a.Map.Elems
		bm := b.Map.Elems
		if len(am) != len(bm) {
			return false
		}
		for k, av := range am {
			bv, ok := bm[k]
			if !ok {
				return false
			}
			if !i.valuesEqual(av, bv) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (i *Interpreter) compareStrings(op string, a string, b string) (bool, bool) {
	switch op {
	case "<":
		return a < b, true
	case ">":
		return a > b, true
	case "<=":
		return a <= b, true
	case ">=":
		return a >= b, true
	default:
		return false, false
	}
}

func (i *Interpreter) toIndex(v Value, span ast.Span) (int, error) {
	if v.Kind != ValNumber {
		return 0, i.runtimeErr(span, "Array index must be a number")
	}
	idx := int(v.Number)
	if v.Number != float64(idx) {
		return 0, i.runtimeErr(span, "Array index must be an integer")
	}
	return idx, nil
}

// ---------- String helpers (runes) ----------

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func substrRunes(s string, start int, length *int) (string, bool) {
	rs := []rune(s)
	if start < 0 || start > len(rs) {
		return "", false
	}
	if length == nil {
		return string(rs[start:]), true
	}
	if *length < 0 {
		return "", false
	}
	end := start + *length
	if end > len(rs) {
		end = len(rs)
	}
	return string(rs[start:end]), true
}

func runeIndexOf(hay, needle string) int {
	hs := []rune(hay)
	ns := []rune(needle)
	if len(ns) == 0 {
		return 0
	}
	if len(ns) > len(hs) {
		return -1
	}
	for i := 0; i <= len(hs)-len(ns); i++ {
		ok := true
		for j := 0; j < len(ns); j++ {
			if hs[i+j] != ns[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func runeLastIndexOf(hay, needle string) int {
	hs := []rune(hay)
	ns := []rune(needle)
	if len(ns) == 0 {
		return len(hs)
	}
	if len(ns) > len(hs) {
		return -1
	}
	for i := len(hs) - len(ns); i >= 0; i-- {
		ok := true
		for j := 0; j < len(ns); j++ {
			if hs[i+j] != ns[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// ---------- Expressions ----------

func (i *Interpreter) evalExpr(e ast.Expr) (Value, error) {
	switch expr := e.(type) {
	case *ast.StringLiteral:
		return StringValue(expr.Value), nil

	case *ast.NumberLiteral:
		n, err := strconv.ParseFloat(expr.Lexeme, 64)
		if err != nil {
			return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Invalid number %q", expr.Lexeme))
		}
		return NumberValue(n), nil

	case *ast.BoolLiteral:
		return BoolValue(expr.Value), nil

	case *ast.ArrayLiteralExpr:
		els := make([]Value, 0, len(expr.Elements))
		for _, el := range expr.Elements {
			v, err := i.evalExpr(el)
			if err != nil {
				return Value{}, err
			}
			els = append(els, v)
		}
		return ArrayValue(els), nil

	case *ast.MapLiteralExpr:
		m := map[string]Value{}
		for _, ent := range expr.Entries {
			v, err := i.evalExpr(ent.Value)
			if err != nil {
				return Value{}, err
			}
			m[ent.Key] = v
		}
		return MapValue(m), nil

	case *ast.IndexExpr:
		left, err := i.evalExpr(expr.Left)
		if err != nil {
			return Value{}, err
		}
		iv, err := i.evalExpr(expr.Index)
		if err != nil {
			return Value{}, err
		}

		if left.Kind == ValArray && left.Arr != nil {
			idx, err := i.toIndex(iv, expr.Index.GetSpan())
			if err != nil {
				return Value{}, err
			}
			if idx < 0 || idx >= len(left.Arr.Elems) {
				return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Array index out of bounds (index %d, size %d)", idx, len(left.Arr.Elems)))
			}
			return left.Arr.Elems[idx], nil
		}

		if left.Kind == ValMap && left.Map != nil {
			if iv.Kind != ValString {
				return Value{}, i.runtimeErr(expr.Index.GetSpan(), "Map key must be a string")
			}
			val, ok := left.Map.Elems[iv.Str]
			if !ok {
				return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Map key %q not found", iv.Str))
			}
			return val, nil
		}

		return Value{}, i.runtimeErr(expr.GetSpan(), "Indexing requires an array or map")

	case *ast.Identifier:
		if i.inFunction() {
			if v, ok := i.currentEnv()[expr.Name]; ok {
				return v, nil
			}
		}
		if v, ok := i.globals[expr.Name]; ok {
			return v, nil
		}
		return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Undefined variable %q", expr.Name))

	case *ast.CallExpr:
		return i.evalCall(expr)

	case *ast.UnaryExpr:
		right, err := i.evalExpr(expr.Right)
		if err != nil {
			return Value{}, err
		}
		switch expr.Op {
		case "not":
			if right.Kind != ValBool {
				return Value{}, i.runtimeErr(expr.GetSpan(), "Operator 'not' requires boolean")
			}
			return BoolValue(!right.Bool), nil
		default:
			return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Unknown unary operator %q", expr.Op))
		}

	case *ast.BinaryExpr:
		if expr.Op == "and" || expr.Op == "or" {
			left, err := i.evalExpr(expr.Left)
			if err != nil {
				return Value{}, err
			}
			if left.Kind != ValBool {
				return Value{}, i.runtimeErr(expr.Left.GetSpan(), fmt.Sprintf("Operator %q requires booleans", expr.Op))
			}

			if expr.Op == "and" {
				if !left.Bool {
					return BoolValue(false), nil
				}
				right, err := i.evalExpr(expr.Right)
				if err != nil {
					return Value{}, err
				}
				if right.Kind != ValBool {
					return Value{}, i.runtimeErr(expr.Right.GetSpan(), "Operator 'and' requires booleans")
				}
				return BoolValue(left.Bool && right.Bool), nil
			}

			if left.Bool {
				return BoolValue(true), nil
			}
			right, err := i.evalExpr(expr.Right)
			if err != nil {
				return Value{}, err
			}
			if right.Kind != ValBool {
				return Value{}, i.runtimeErr(expr.Right.GetSpan(), "Operator 'or' requires booleans")
			}
			return BoolValue(left.Bool || right.Bool), nil
		}

		left, err := i.evalExpr(expr.Left)
		if err != nil {
			return Value{}, err
		}
		right, err := i.evalExpr(expr.Right)
		if err != nil {
			return Value{}, err
		}

		if expr.Op == "+" {
			if left.Kind == ValNumber && right.Kind == ValNumber {
				return NumberValue(left.Number + right.Number), nil
			}
			if left.Kind == ValArray && right.Kind == ValArray {
				if left.Arr == nil || right.Arr == nil {
					return Value{}, i.runtimeErr(expr.GetSpan(), "Array concat requires valid arrays")
				}
				out := make([]Value, 0, len(left.Arr.Elems)+len(right.Arr.Elems))
				out = append(out, left.Arr.Elems...)
				out = append(out, right.Arr.Elems...)
				return ArrayValue(out), nil
			}
			return StringValue(left.ToString() + right.ToString()), nil
		}

		if expr.Op == "==" || expr.Op == "!=" {
			eq := i.valuesEqual(left, right)
			if expr.Op == "!=" {
				eq = !eq
			}
			return BoolValue(eq), nil
		}

		if expr.Op == "<" || expr.Op == ">" || expr.Op == "<=" || expr.Op == ">=" {
			if left.Kind == ValNumber && right.Kind == ValNumber {
				switch expr.Op {
				case "<":
					return BoolValue(left.Number < right.Number), nil
				case ">":
					return BoolValue(left.Number > right.Number), nil
				case "<=":
					return BoolValue(left.Number <= right.Number), nil
				case ">=":
					return BoolValue(left.Number >= right.Number), nil
				}
			}
			if left.Kind == ValString && right.Kind == ValString {
				res, ok := i.compareStrings(expr.Op, left.Str, right.Str)
				if !ok {
					return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Unknown operator %q", expr.Op))
				}
				return BoolValue(res), nil
			}
			return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Operator %q requires two numbers or two strings", expr.Op))
		}

		if left.Kind != ValNumber || right.Kind != ValNumber {
			return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Operator %q requires numbers", expr.Op))
		}

		switch expr.Op {
		case "-":
			return NumberValue(left.Number - right.Number), nil
		case "*":
			return NumberValue(left.Number * right.Number), nil
		case "/":
			return NumberValue(left.Number / right.Number), nil
		}

		return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Unknown operator %q", expr.Op))

	default:
		span, ok := ast.SpanOf(e)
		if !ok {
			span = ast.Span{}
		}
		return Value{}, i.runtimeErr(span, "Unsupported expression")
	}
}

func (i *Interpreter) evalCall(call *ast.CallExpr) (Value, error) {
	if fn, ok := i.funcs[call.Callee]; ok {
		return i.evalUserCall(fn, call.Args, call.GetSpan())
	}
	return i.evalBuiltin(call.Callee, call.Args, call.GetSpan())
}

func (i *Interpreter) evalUserCall(fn *ast.FunctionDecl, args []ast.Expr, callSpan ast.Span) (Value, error) {
	if len(args) != len(fn.Params) {
		return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("Function %q expects %d args, got %d", fn.Name, len(fn.Params), len(args)))
	}

	argVals := []Value{}
	for _, a := range args {
		v, err := i.evalExpr(a)
		if err != nil {
			return Value{}, err
		}
		argVals = append(argVals, v)
	}

	i.callStack = append(i.callStack, fn.Name)
	i.pushLocals()
	defer func() {
		i.popLocals()
		i.callStack = i.callStack[:len(i.callStack)-1]
	}()

	for idx, name := range fn.Params {
		i.currentEnv()[name] = argVals[idx]
	}

	err := i.Run(fn.Body)
	if rs, ok := err.(ReturnSignal); ok {
		return rs.Val, nil
	}
	if err != nil {
		return Value{}, err
	}
	return Value{}, i.runtimeErr(fn.GetSpan(), fmt.Sprintf("Function %q ended without return", fn.Name))
}

// ---------- Builtins ----------

func (i *Interpreter) getHandleReader(handle int) (*bufio.Reader, *os.File, error) {
	f, ok := i.files[handle]
	if !ok || f == nil {
		return nil, nil, fmt.Errorf("handle #%d is not open", handle)
	}
	if r, ok := i.readers[handle]; ok && r != nil {
		return r, f, nil
	}
	r := bufio.NewReader(f)
	i.readers[handle] = r
	return r, f, nil
}

func (i *Interpreter) evalBuiltin(name string, argExprs []ast.Expr, callSpan ast.Span) (Value, error) {
	args := []Value{}
	for _, a := range argExprs {
		v, err := i.evalExpr(a)
		if err != nil {
			return Value{}, err
		}
		args = append(args, v)
	}

	switch name {
	// --- core ---
	case "str":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "str() expects 1 arg")
		}
		return StringValue(args[0].ToString()), nil

	case "num":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "num() expects 1 arg")
		}
		if args[0].Kind == ValNumber {
			return args[0], nil
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(args[0].ToString()), 64)
		if err != nil {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("num() could not parse %q", args[0].ToString()))
		}
		return NumberValue(n), nil

	case "len":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "len() expects 1 arg")
		}
		switch args[0].Kind {
		case ValString:
			return NumberValue(float64(runeLen(args[0].Str))), nil
		case ValArray:
			if args[0].Arr == nil {
				return NumberValue(0), nil
			}
			return NumberValue(float64(len(args[0].Arr.Elems))), nil
		case ValMap:
			if args[0].Map == nil || args[0].Map.Elems == nil {
				return NumberValue(0), nil
			}
			return NumberValue(float64(len(args[0].Map.Elems))), nil
		default:
			return Value{}, i.runtimeErr(callSpan, "len() expects a string, array, or map")
		}

	// --- string funcs ---
	case "lower":
		if len(args) != 1 || args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "lower() expects 1 string arg")
		}
		return StringValue(strings.ToLower(args[0].Str)), nil

	case "upper":
		if len(args) != 1 || args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "upper() expects 1 string arg")
		}
		return StringValue(strings.ToUpper(args[0].Str)), nil

	case "trim":
		if len(args) != 1 && len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "trim() expects 1 or 2 args: trim(s [,cutset])")
		}
		if args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "trim() first arg must be a string")
		}
		if len(args) == 1 {
			return StringValue(strings.TrimSpace(args[0].Str)), nil
		}
		if args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "trim() cutset must be a string")
		}
		return StringValue(strings.Trim(args[0].Str, args[1].Str)), nil

	case "ltrim":
		if len(args) != 1 && len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "ltrim() expects 1 or 2 args: ltrim(s [,cutset])")
		}
		if args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "ltrim() first arg must be a string")
		}
		if len(args) == 1 {
			return StringValue(strings.TrimLeftFunc(args[0].Str, unicode.IsSpace)), nil
		}
		if args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "ltrim() cutset must be a string")
		}
		return StringValue(strings.TrimLeft(args[0].Str, args[1].Str)), nil

	case "rtrim":
		if len(args) != 1 && len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "rtrim() expects 1 or 2 args: rtrim(s [,cutset])")
		}
		if args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "rtrim() first arg must be a string")
		}
		if len(args) == 1 {
			return StringValue(strings.TrimRightFunc(args[0].Str, unicode.IsSpace)), nil
		}
		if args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "rtrim() cutset must be a string")
		}
		return StringValue(strings.TrimRight(args[0].Str, args[1].Str)), nil

	case "contains":
		if len(args) != 2 || args[0].Kind != ValString || args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "contains() expects 2 string args: contains(s, sub)")
		}
		return BoolValue(strings.Contains(args[0].Str, args[1].Str)), nil

	case "startswith":
		if len(args) != 2 || args[0].Kind != ValString || args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "startswith() expects 2 string args: startswith(s, prefix)")
		}
		return BoolValue(strings.HasPrefix(args[0].Str, args[1].Str)), nil

	case "endswith":
		if len(args) != 2 || args[0].Kind != ValString || args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "endswith() expects 2 string args: endswith(s, suffix)")
		}
		return BoolValue(strings.HasSuffix(args[0].Str, args[1].Str)), nil

	case "replace":
		if len(args) != 3 && len(args) != 4 {
			return Value{}, i.runtimeErr(callSpan, "replace() expects 3 or 4 args: replace(s, old, new [,n])")
		}
		if args[0].Kind != ValString || args[1].Kind != ValString || args[2].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "replace() expects string args for s/old/new")
		}
		n := -1
		if len(args) == 4 {
			if args[3].Kind != ValNumber {
				return Value{}, i.runtimeErr(callSpan, "replace() n must be a number")
			}
			n = int(args[3].Number)
		}
		return StringValue(strings.Replace(args[0].Str, args[1].Str, args[2].Str, n)), nil

	case "split":
		if len(args) != 2 || args[0].Kind != ValString || args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "split() expects 2 string args: split(s, sep)")
		}
		parts := strings.Split(args[0].Str, args[1].Str)
		out := make([]Value, 0, len(parts))
		for _, p := range parts {
			out = append(out, StringValue(p))
		}
		return ArrayValue(out), nil

	case "join":
		if len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "join() expects 2 args: join(array, sep)")
		}
		if args[0].Kind != ValArray || args[0].Arr == nil {
			return Value{}, i.runtimeErr(callSpan, "join() first arg must be an array")
		}
		if args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "join() sep must be a string")
		}
		sep := args[1].Str
		elems := args[0].Arr.Elems
		ss := make([]string, 0, len(elems))
		for _, v := range elems {
			ss = append(ss, v.ToString())
		}
		return StringValue(strings.Join(ss, sep)), nil

	case "indexof":
		if len(args) != 2 || args[0].Kind != ValString || args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "indexof() expects 2 string args: indexof(s, sub)")
		}
		return NumberValue(float64(runeIndexOf(args[0].Str, args[1].Str))), nil

	case "lastindexof":
		if len(args) != 2 || args[0].Kind != ValString || args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "lastindexof() expects 2 string args: lastindexof(s, sub)")
		}
		return NumberValue(float64(runeLastIndexOf(args[0].Str, args[1].Str))), nil

	case "repeat":
		if len(args) != 2 || args[0].Kind != ValString || args[1].Kind != ValNumber {
			return Value{}, i.runtimeErr(callSpan, "repeat() expects (string, number): repeat(s, n)")
		}
		n := int(args[1].Number)
		if n < 0 {
			return Value{}, i.runtimeErr(callSpan, "repeat() n must be >= 0")
		}
		return StringValue(strings.Repeat(args[0].Str, n)), nil

	case "substr":
		if len(args) != 2 && len(args) != 3 {
			return Value{}, i.runtimeErr(callSpan, "substr() expects 2 or 3 args: substr(s, start [,len])")
		}
		if args[0].Kind != ValString || args[1].Kind != ValNumber {
			return Value{}, i.runtimeErr(callSpan, "substr() expects (string, number [,number])")
		}
		start := int(args[1].Number)
		if args[1].Number != float64(start) {
			return Value{}, i.runtimeErr(callSpan, "substr() start must be an integer")
		}
		if len(args) == 2 {
			out, ok := substrRunes(args[0].Str, start, nil)
			if !ok {
				return Value{}, i.runtimeErr(callSpan, "substr() out of range")
			}
			return StringValue(out), nil
		}
		if args[2].Kind != ValNumber {
			return Value{}, i.runtimeErr(callSpan, "substr() len must be a number")
		}
		l := int(args[2].Number)
		if args[2].Number != float64(l) {
			return Value{}, i.runtimeErr(callSpan, "substr() len must be an integer")
		}
		out, ok := substrRunes(args[0].Str, start, &l)
		if !ok {
			return Value{}, i.runtimeErr(callSpan, "substr() out of range")
		}
		return StringValue(out), nil

	// --- file handle read helpers ---
	case "lineinput":
		// lineinput(handle) -> string | null
		if len(args) != 1 || args[0].Kind != ValNumber {
			return Value{}, i.runtimeErr(callSpan, "lineinput() expects 1 number arg: lineinput(handle)")
		}
		h := int(args[0].Number)
		if args[0].Number != float64(h) || h <= 0 {
			return Value{}, i.runtimeErr(callSpan, "lineinput() handle must be a positive integer")
		}

		r, _, herr := i.getHandleReader(h)
		if herr != nil {
			return Value{}, i.runtimeErr(callSpan, "lineinput() failed: "+herr.Error())
		}

		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if line == "" {
					return NullValue(), nil
				}
				// return last partial line
				return StringValue(strings.TrimRight(line, "\r\n")), nil
			}
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("lineinput() failed: %v", err))
		}
		return StringValue(strings.TrimRight(line, "\r\n")), nil

	case "eof":
		// eof(handle) -> bool
		if len(args) != 1 || args[0].Kind != ValNumber {
			return Value{}, i.runtimeErr(callSpan, "eof() expects 1 number arg: eof(handle)")
		}
		h := int(args[0].Number)
		if args[0].Number != float64(h) || h <= 0 {
			return Value{}, i.runtimeErr(callSpan, "eof() handle must be a positive integer")
		}

		// per your preference: if not open, treat as EOF=true
		if _, ok := i.files[h]; !ok || i.files[h] == nil {
			return BoolValue(true), nil
		}

		r, _, herr := i.getHandleReader(h)
		if herr != nil {
			return BoolValue(true), nil
		}

		_, err := r.Peek(1)
		if err == io.EOF {
			return BoolValue(true), nil
		}
		if err != nil {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("eof() failed: %v", err))
		}
		return BoolValue(false), nil

	// Input (stdin)
	case "input":
		if len(args) > 1 {
			return Value{}, i.runtimeErr(callSpan, "input() expects 0 or 1 args")
		}
		if len(args) == 1 {
			fmt.Print(args[0].ToString())
		}
		line, _ := i.in.ReadString('\n')
		return StringValue(strings.TrimRight(line, "\r\n")), nil
	}

	return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("Undefined function %q", name))
}
