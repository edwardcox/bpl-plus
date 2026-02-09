package interpreter

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"bpl-plus/ast"
)

type ValueKind int

const (
	ValNull ValueKind = iota
	ValNumber
	ValString
	ValBool
	ValArray
)

// ArrayObject gives arrays reference semantics.
// Copying a Value copies the pointer, so mutations affect all references.
type ArrayObject struct {
	Elems []Value
}

type Value struct {
	Kind   ValueKind
	Number float64
	Str    string
	Bool   bool
	Arr    *ArrayObject
}

func NullValue() Value            { return Value{Kind: ValNull} }
func NumberValue(n float64) Value { return Value{Kind: ValNumber, Number: n} }
func StringValue(s string) Value  { return Value{Kind: ValString, Str: s} }
func BoolValue(b bool) Value      { return Value{Kind: ValBool, Bool: b} }
func ArrayValue(elems []Value) Value {
	return Value{Kind: ValArray, Arr: &ArrayObject{Elems: elems}}
}

func (v Value) arrayElems() []Value {
	if v.Kind != ValArray || v.Arr == nil {
		return nil
	}
	return v.Arr.Elems
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
	default:
		return "null"
	}
}

type ReturnSignal struct {
	Val Value
}

func (r ReturnSignal) Error() string { return "return" }

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

type Interpreter struct {
	globals map[string]Value
	locals  []map[string]Value
	funcs   map[string]*ast.FunctionDecl

	in *bufio.Reader

	filename string
	lines    []string

	callStack []string
}

func NewWithSource(filename string, source string) *Interpreter {
	return &Interpreter{
		globals:   map[string]Value{},
		locals:    []map[string]Value{},
		funcs:     map[string]*ast.FunctionDecl{},
		in:        bufio.NewReader(os.Stdin),
		filename:  filename,
		lines:     splitLinesPreserve(source),
		callStack: []string{},
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
			if _, ok := err.(ReturnSignal); ok {
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
			if err := i.Run(stmt.Body); err != nil {
				return err
			}
		}
		return nil

	case *ast.ForStmt:
		return i.execFor(stmt)

	case *ast.ForEachStmt:
		iterVal, err := i.evalExpr(stmt.Iterable)
		if err != nil {
			return err
		}
		if iterVal.Kind != ValArray || iterVal.Arr == nil {
			return i.runtimeErr(stmt.GetSpan(), "foreach requires an array on the right side of 'in'")
		}

		// Snapshot for predictable behavior if the array is mutated during iteration.
		elems := iterVal.Arr.Elems
		for idx, el := range elems {
			i.currentEnv()[stmt.Var] = el
			if stmt.IndexVar != "" {
				i.currentEnv()[stmt.IndexVar] = NumberValue(float64(idx))
			}
			if err := i.Run(stmt.Body); err != nil {
				return err
			}
		}
		return nil

	default:
		span, ok := ast.SpanOf(s)
		if !ok {
			span = ast.Span{}
		}
		return i.runtimeErr(span, fmt.Sprintf("Unsupported statement %s", s.NodeKind()))
	}
}

func (i *Interpreter) execIndexAssign(stmt *ast.IndexAssignStmt) error {
	env, arrVal, ok := i.findVarEnv(stmt.Name)
	if !ok {
		return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("Undefined variable %q", stmt.Name))
	}
	if arrVal.Kind != ValArray || arrVal.Arr == nil {
		return i.runtimeErr(stmt.GetSpan(), "Index assignment requires an array")
	}

	iv, err := i.evalExpr(stmt.Index)
	if err != nil {
		return err
	}
	idx, err := i.toIndex(iv, stmt.Index.GetSpan())
	if err != nil {
		return err
	}

	elems := arrVal.Arr.Elems
	if idx < 0 || idx >= len(elems) {
		return i.runtimeErr(stmt.GetSpan(), fmt.Sprintf("Array index out of bounds (index %d, size %d)", idx, len(elems)))
	}

	newVal, err := i.evalExpr(stmt.Value)
	if err != nil {
		return err
	}

	arrVal.Arr.Elems[idx] = newVal
	env[stmt.Name] = arrVal
	return nil
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
		if err := i.Run(stmt.Body); err != nil {
			return err
		}
		i.currentEnv()[stmt.Var] = NumberValue(cur + step)
	}

	return nil
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

	case *ast.IndexExpr:
		left, err := i.evalExpr(expr.Left)
		if err != nil {
			return Value{}, err
		}
		if left.Kind != ValArray || left.Arr == nil {
			return Value{}, i.runtimeErr(expr.GetSpan(), "Indexing requires an array")
		}
		iv, err := i.evalExpr(expr.Index)
		if err != nil {
			return Value{}, err
		}
		idx, err := i.toIndex(iv, expr.Index.GetSpan())
		if err != nil {
			return Value{}, err
		}
		if idx < 0 || idx >= len(left.Arr.Elems) {
			return Value{}, i.runtimeErr(expr.GetSpan(), fmt.Sprintf("Array index out of bounds (index %d, size %d)", idx, len(left.Arr.Elems)))
		}
		return left.Arr.Elems[idx], nil

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
		// short-circuit booleans
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
			// number + number
			if left.Kind == ValNumber && right.Kind == ValNumber {
				return NumberValue(left.Number + right.Number), nil
			}

			// array + array (concat, non-mutating)
			if left.Kind == ValArray && right.Kind == ValArray {
				if left.Arr == nil || right.Arr == nil {
					return Value{}, i.runtimeErr(expr.GetSpan(), "Array concat requires valid arrays")
				}
				out := make([]Value, 0, len(left.Arr.Elems)+len(right.Arr.Elems))
				out = append(out, left.Arr.Elems...)
				out = append(out, right.Arr.Elems...)
				return ArrayValue(out), nil
			}

			// fallback: string concat
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
		default:
			return Value{}, i.runtimeErr(callSpan, "len() expects a string or array")
		}

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
			return Value{}, i.runtimeErr(callSpan, "pop() argument must be an array")
		}
		elems := args[0].Arr.Elems
		if len(elems) == 0 {
			return Value{}, i.runtimeErr(callSpan, "pop() cannot pop from an empty array")
		}
		removed := elems[len(elems)-1]
		args[0].Arr.Elems = elems[:len(elems)-1]
		return removed, nil

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
		elems := args[0].Arr.Elems
		if idx < 0 || idx > len(elems) {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("insert() index out of bounds (index %d, size %d)", idx, len(elems)))
		}
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
		elems := args[0].Arr.Elems
		if idx < 0 || idx >= len(elems) {
			return Value{}, i.runtimeErr(callSpan, fmt.Sprintf("remove() index out of bounds (index %d, size %d)", idx, len(elems)))
		}
		removed := elems[idx]
		copy(elems[idx:], elems[idx+1:])
		elems = elems[:len(elems)-1]
		args[0].Arr.Elems = elems
		return removed, nil

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
