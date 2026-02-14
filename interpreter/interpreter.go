package interpreter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
// Copying a Value copies the pointer, so mutations affect all references.
type ArrayObject struct {
	Elems []Value
}

// MapObject gives maps reference semantics.
// Copying a Value copies the pointer, so mutations affect all references.
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

type ReturnSignal struct {
	Val Value
}

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

	// Loop tracking (for validating break/continue usage)
	loopDepth int

	// Modules: resolved absolute-ish path -> state (loading/loaded)
	modules map[string]moduleState

	// Stack of currently importing modules for circular import diagnostics
	moduleStack []string
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
		loopDepth:   0,
		modules:     map[string]moduleState{},
		moduleStack: []string{},
	}
}

func New() *Interpreter {
	return NewWithSource("", "")
}

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
			// Propagate control-flow signals upward so loops can catch them.
			if _, ok := err.(ReturnSignal); ok {
				return err
			}
			if _, ok := err.(BreakSignal); ok {
				return err
			}
			if _, ok := err.(ContinueSignal); ok {
				return err
			}
			return err
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
		if i.loopDepth <= 0 {
			return i.runtimeErr(stmt.GetSpan(), "break is only valid inside a loop")
		}
		return BreakSignal{}

	case *ast.ContinueStmt:
		if i.loopDepth <= 0 {
			return i.runtimeErr(stmt.GetSpan(), "continue is only valid inside a loop")
		}
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
		i.loopDepth++
		defer func() { i.loopDepth-- }()

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
				case ReturnSignal:
					return err
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

	i.loopDepth++
	defer func() { i.loopDepth-- }()

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
			case ReturnSignal:
				return err
			default:
				return err
			}
		}

		i.currentEnv()[stmt.Var] = NumberValue(cur + step)
	}

	return nil
}

func (i *Interpreter) execForEach(stmt *ast.ForEachStmt) error {
	iter, err := i.evalExpr(stmt.Iterable)
	if err != nil {
		return err
	}

	i.loopDepth++
	defer func() { i.loopDepth-- }()

	// foreach over array
	if iter.Kind == ValArray && iter.Arr != nil {
		for idx, v := range iter.Arr.Elems {
			i.currentEnv()[stmt.Var] = v
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
				case ReturnSignal:
					return err
				default:
					return err
				}
			}
		}
		return nil
	}

	// foreach over map (stable key order)
	if iter.Kind == ValMap && iter.Map != nil {
		m := iter.Map.Elems
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for idx, k := range keys {
			// VALUE var gets key (matches your demo output: a b c)
			i.currentEnv()[stmt.Var] = StringValue(k)
			if stmt.IndexVar != "" {
				// INDEX var gets value (matches your demo output: a=1 etc when you print key=value)
				i.currentEnv()[stmt.IndexVar] = m[k]
			}

			_ = idx // idx intentionally unused here (kept in case you want numeric index later)

			err := i.Run(stmt.Body)
			if err != nil {
				switch err.(type) {
				case BreakSignal:
					return nil
				case ContinueSignal:
					continue
				case ReturnSignal:
					return err
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
			return NumberValue(float64(len([]rune(args[0].Str)))), nil
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

	// Maps helpers
	case "has":
		if len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "has() expects 2 args: has(map, key)")
		}
		if args[0].Kind != ValMap || args[0].Map == nil {
			return Value{}, i.runtimeErr(callSpan, "has() first argument must be a map")
		}
		if args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "has() key must be a string")
		}
		_, ok := args[0].Map.Elems[args[1].Str]
		return BoolValue(ok), nil

	case "get":
		if len(args) != 3 {
			return Value{}, i.runtimeErr(callSpan, "get() expects 3 args: get(map, key, default)")
		}
		if args[0].Kind != ValMap || args[0].Map == nil {
			return Value{}, i.runtimeErr(callSpan, "get() first argument must be a map")
		}
		if args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "get() key must be a string")
		}
		if v, ok := args[0].Map.Elems[args[1].Str]; ok {
			return v, nil
		}
		return args[2], nil

	case "keys":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "keys() expects 1 arg: keys(map)")
		}
		if args[0].Kind != ValMap || args[0].Map == nil {
			return Value{}, i.runtimeErr(callSpan, "keys() expects a map")
		}
		m := args[0].Map.Elems
		ks := make([]string, 0, len(m))
		for k := range m {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		out := make([]Value, 0, len(ks))
		for _, k := range ks {
			out = append(out, StringValue(k))
		}
		return ArrayValue(out), nil

	case "values":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "values() expects 1 arg: values(map)")
		}
		if args[0].Kind != ValMap || args[0].Map == nil {
			return Value{}, i.runtimeErr(callSpan, "values() expects a map")
		}
		m := args[0].Map.Elems
		ks := make([]string, 0, len(m))
		for k := range m {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		out := make([]Value, 0, len(ks))
		for _, k := range ks {
			out = append(out, m[k])
		}
		return ArrayValue(out), nil

	// Maps Step C
	case "items":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "items() expects 1 arg: items(map)")
		}
		if args[0].Kind != ValMap || args[0].Map == nil {
			return Value{}, i.runtimeErr(callSpan, "items() expects a map")
		}
		m := args[0].Map.Elems
		if m == nil {
			return ArrayValue([]Value{}), nil
		}

		ks := make([]string, 0, len(m))
		for k := range m {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		out := make([]Value, 0, len(ks))
		for _, k := range ks {
			pair := ArrayValue([]Value{StringValue(k), m[k]})
			out = append(out, pair)
		}
		return ArrayValue(out), nil

	case "del":
		if len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "del() expects 2 args: del(map, key)")
		}
		if args[0].Kind != ValMap || args[0].Map == nil {
			return Value{}, i.runtimeErr(callSpan, "del() first argument must be a map")
		}
		if args[1].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "del() key must be a string")
		}
		if args[0].Map.Elems != nil {
			delete(args[0].Map.Elems, args[1].Str)
		}
		return NullValue(), nil

	case "clear":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "clear() expects 1 arg: clear(map)")
		}
		if args[0].Kind != ValMap || args[0].Map == nil {
			return Value{}, i.runtimeErr(callSpan, "clear() expects a map")
		}
		if args[0].Map.Elems == nil {
			args[0].Map.Elems = map[string]Value{}
		} else {
			for k := range args[0].Map.Elems {
				delete(args[0].Map.Elems, k)
			}
		}
		return NullValue(), nil

	// Arrays helpers
	case "push":
		if len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "push() expects 2 arguments: push(array, value)")
		}
		if args[0].Kind != ValArray || args[0].Arr == nil {
			return Value{}, i.runtimeErr(callSpan, "push() first argument must be an array")
		}
		args[0].Arr.Elems = append(args[0].Arr.Elems, args[1])
		return NullValue(), nil

	case "pop":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "pop() expects 1 argument: pop(array)")
		}
		if args[0].Kind != ValArray || args[0].Arr == nil {
			return Value{}, i.runtimeErr(callSpan, "pop() first argument must be an array")
		}
		if len(args[0].Arr.Elems) == 0 {
			return Value{}, i.runtimeErr(callSpan, "pop() cannot pop from an empty array")
		}
		last := args[0].Arr.Elems[len(args[0].Arr.Elems)-1]
		args[0].Arr.Elems = args[0].Arr.Elems[:len(args[0].Arr.Elems)-1]
		return last, nil

	case "insert":
		if len(args) != 3 {
			return Value{}, i.runtimeErr(callSpan, "insert() expects 3 arguments: insert(array, index, value)")
		}
		if args[0].Kind != ValArray || args[0].Arr == nil {
			return Value{}, i.runtimeErr(callSpan, "insert() first argument must be an array")
		}
		idx, err := i.toIndex(args[1], callSpan)
		if err != nil {
			return Value{}, err
		}
		if idx < 0 || idx > len(args[0].Arr.Elems) {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("insert() index out of bounds (index %d, size %d)", idx, len(args[0].Arr.Elems)))
		}
		elems := args[0].Arr.Elems
		elems = append(elems, NullValue())
		copy(elems[idx+1:], elems[idx:])
		elems[idx] = args[2]
		args[0].Arr.Elems = elems
		return NullValue(), nil

	case "remove":
		if len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "remove() expects 2 arguments: remove(array, index)")
		}
		if args[0].Kind != ValArray || args[0].Arr == nil {
			return Value{}, i.runtimeErr(callSpan, "remove() first argument must be an array")
		}
		idx, err := i.toIndex(args[1], callSpan)
		if err != nil {
			return Value{}, err
		}
		if idx < 0 || idx >= len(args[0].Arr.Elems) {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("remove() index out of bounds (index %d, size %d)", idx, len(args[0].Arr.Elems)))
		}
		removed := args[0].Arr.Elems[idx]
		args[0].Arr.Elems = append(args[0].Arr.Elems[:idx], args[0].Arr.Elems[idx+1:]...)
		return removed, nil

	// Input
	case "input":
		if len(args) > 1 {
			return Value{}, i.runtimeErr(callSpan, "input() expects 0 or 1 args")
		}
		if len(args) == 1 {
			fmt.Print(args[0].ToString())
		}
		line, _ := i.in.ReadString('\n')
		return StringValue(strings.TrimRight(line, "\r\n")), nil

	// File I/O
	case "writefile":
		if len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "writefile() expects 2 args: writefile(path, contents)")
		}
		if args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "writefile() path must be a string")
		}
		if err := os.WriteFile(args[0].Str, []byte(args[1].ToString()), 0644); err != nil {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("writefile() failed: %v", err))
		}
		return NullValue(), nil

	case "appendfile":
		if len(args) != 2 {
			return Value{}, i.runtimeErr(callSpan, "appendfile() expects 2 args: appendfile(path, contents)")
		}
		if args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "appendfile() path must be a string")
		}
		f, err := os.OpenFile(args[0].Str, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("appendfile() failed: %v", err))
		}
		defer f.Close()
		if _, err := f.WriteString(args[1].ToString()); err != nil {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("appendfile() failed: %v", err))
		}
		return NullValue(), nil

	case "readfile":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "readfile() expects 1 arg: readfile(path)")
		}
		if args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "readfile() path must be a string")
		}
		data, err := os.ReadFile(args[0].Str)
		if err != nil {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("readfile() failed: %v", err))
		}
		return StringValue(string(data)), nil

	case "exists":
		if len(args) != 1 {
			return Value{}, i.runtimeErr(callSpan, "exists() expects 1 arg: exists(path)")
		}
		if args[0].Kind != ValString {
			return Value{}, i.runtimeErr(callSpan, "exists() path must be a string")
		}
		_, err := os.Stat(args[0].Str)
		if err == nil {
			return BoolValue(true), nil
		}
		if os.IsNotExist(err) {
			return BoolValue(false), nil
		}
		return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("exists() failed: %v", err))
	}

	return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("Undefined function %q", name))
}
