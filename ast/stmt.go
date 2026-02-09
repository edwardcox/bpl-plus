package ast

import "fmt"

type Stmt interface {
	Node
	stmtNode()
	String() string
	GetSpan() Span
}

type PrintStmt struct {
	S     Span
	Value Expr
}

func (p *PrintStmt) NodeKind() string { return "PrintStmt" }
func (p *PrintStmt) stmtNode()        {}
func (p *PrintStmt) GetSpan() Span    { return p.S }
func (p *PrintStmt) String() string   { return fmt.Sprintf("PrintStmt(%s)", p.Value.String()) }

type AssignStmt struct {
	S     Span
	Name  string
	Value Expr
}

func (a *AssignStmt) NodeKind() string { return "AssignStmt" }
func (a *AssignStmt) stmtNode()        {}
func (a *AssignStmt) GetSpan() Span    { return a.S }
func (a *AssignStmt) String() string {
	return fmt.Sprintf("AssignStmt(%s = %s)", a.Name, a.Value.String())
}

// --- Arrays v1 (index assignment) ---
// a[i] = value
type IndexAssignStmt struct {
	S     Span
	Name  string
	Index Expr
	Value Expr
}

func (x *IndexAssignStmt) NodeKind() string { return "IndexAssignStmt" }
func (x *IndexAssignStmt) stmtNode()        {}
func (x *IndexAssignStmt) GetSpan() Span    { return x.S }
func (x *IndexAssignStmt) String() string {
	return fmt.Sprintf("IndexAssign(%s[%s] = %s)", x.Name, x.Index.String(), x.Value.String())
}

// --- Expression statements ---
// e.g. push(a, 1)
type ExprStmt struct {
	S    Span
	Expr Expr
}

func (e *ExprStmt) NodeKind() string { return "ExprStmt" }
func (e *ExprStmt) stmtNode()        {}
func (e *ExprStmt) GetSpan() Span    { return e.S }
func (e *ExprStmt) String() string   { return fmt.Sprintf("ExprStmt(%s)", e.Expr.String()) }

type IfStmt struct {
	S         Span
	Condition Expr
	Then      []Stmt
	Else      []Stmt
}

func (i *IfStmt) NodeKind() string { return "IfStmt" }
func (i *IfStmt) stmtNode()        {}
func (i *IfStmt) GetSpan() Span    { return i.S }
func (i *IfStmt) String() string {
	return fmt.Sprintf("IfStmt(%s, then=%d, else=%d)", i.Condition.String(), len(i.Then), len(i.Else))
}

type WhileStmt struct {
	S         Span
	Condition Expr
	Body      []Stmt
}

func (w *WhileStmt) NodeKind() string { return "WhileStmt" }
func (w *WhileStmt) stmtNode()        {}
func (w *WhileStmt) GetSpan() Span    { return w.S }
func (w *WhileStmt) String() string {
	return fmt.Sprintf("WhileStmt(%s, body=%d)", w.Condition.String(), len(w.Body))
}

type ForStmt struct {
	S     Span
	Var   string
	Start Expr
	End   Expr
	Step  Expr // optional (nil means default)
	Body  []Stmt
}

func (f *ForStmt) NodeKind() string { return "ForStmt" }
func (f *ForStmt) stmtNode()        {}
func (f *ForStmt) GetSpan() Span    { return f.S }
func (f *ForStmt) String() string {
	step := "nil"
	if f.Step != nil {
		step = f.Step.String()
	}
	return fmt.Sprintf("ForStmt(%s=%s to %s step %s, body=%d)", f.Var, f.Start.String(), f.End.String(), step, len(f.Body))
}

// --- ForEach sugar ---
// for each x in expr
// for each x, i in expr
type ForEachStmt struct {
	S        Span
	Var      string
	IndexVar string // optional ("" means not provided)
	Iterable Expr
	Body     []Stmt
}

func (f *ForEachStmt) NodeKind() string { return "ForEachStmt" }
func (f *ForEachStmt) stmtNode()        {}
func (f *ForEachStmt) GetSpan() Span    { return f.S }
func (f *ForEachStmt) String() string {
	if f.IndexVar != "" {
		return fmt.Sprintf("ForEachStmt(%s,%s in %s, body=%d)", f.Var, f.IndexVar, f.Iterable.String(), len(f.Body))
	}
	return fmt.Sprintf("ForEachStmt(%s in %s, body=%d)", f.Var, f.Iterable.String(), len(f.Body))
}

type FunctionDecl struct {
	S      Span
	Name   string
	Params []string
	Body   []Stmt
}

func (f *FunctionDecl) NodeKind() string { return "FunctionDecl" }
func (f *FunctionDecl) stmtNode()        {}
func (f *FunctionDecl) GetSpan() Span    { return f.S }
func (f *FunctionDecl) String() string {
	return fmt.Sprintf("Function(%s, params=%d, body=%d)", f.Name, len(f.Params), len(f.Body))
}

type ReturnStmt struct {
	S     Span
	Value Expr
}

func (r *ReturnStmt) NodeKind() string { return "ReturnStmt" }
func (r *ReturnStmt) stmtNode()        {}
func (r *ReturnStmt) GetSpan() Span    { return r.S }
func (r *ReturnStmt) String() string   { return fmt.Sprintf("Return(%s)", r.Value.String()) }
