package ast

import (
	"fmt"
	"strings"
)

type Expr interface {
	Node
	exprNode()
	String() string
	GetSpan() Span
}

type StringLiteral struct {
	S     Span
	Value string
}

func (s *StringLiteral) NodeKind() string { return "StringLiteral" }
func (s *StringLiteral) exprNode()        {}
func (s *StringLiteral) GetSpan() Span    { return s.S }
func (s *StringLiteral) String() string   { return fmt.Sprintf("String(%q)", s.Value) }

type NumberLiteral struct {
	S      Span
	Lexeme string
}

func (n *NumberLiteral) NodeKind() string { return "NumberLiteral" }
func (n *NumberLiteral) exprNode()        {}
func (n *NumberLiteral) GetSpan() Span    { return n.S }
func (n *NumberLiteral) String() string   { return fmt.Sprintf("Number(%s)", n.Lexeme) }

type BoolLiteral struct {
	S     Span
	Value bool
}

func (b *BoolLiteral) NodeKind() string { return "BoolLiteral" }
func (b *BoolLiteral) exprNode()        {}
func (b *BoolLiteral) GetSpan() Span    { return b.S }
func (b *BoolLiteral) String() string {
	if b.Value {
		return "Bool(true)"
	}
	return "Bool(false)"
}

type Identifier struct {
	S    Span
	Name string
}

func (i *Identifier) NodeKind() string { return "Identifier" }
func (i *Identifier) exprNode()        {}
func (i *Identifier) GetSpan() Span    { return i.S }
func (i *Identifier) String() string   { return fmt.Sprintf("Ident(%s)", i.Name) }

type UnaryExpr struct {
	S     Span
	Op    string
	Right Expr
}

func (u *UnaryExpr) NodeKind() string { return "UnaryExpr" }
func (u *UnaryExpr) exprNode()        {}
func (u *UnaryExpr) GetSpan() Span    { return u.S }
func (u *UnaryExpr) String() string {
	return fmt.Sprintf("Unary(%s %s)", u.Op, u.Right.String())
}

type BinaryExpr struct {
	S     Span
	Left  Expr
	Op    string
	Right Expr
}

func (b *BinaryExpr) NodeKind() string { return "BinaryExpr" }
func (b *BinaryExpr) exprNode()        {}
func (b *BinaryExpr) GetSpan() Span    { return b.S }
func (b *BinaryExpr) String() string {
	return fmt.Sprintf("Binary(%s %s %s)", b.Left.String(), b.Op, b.Right.String())
}

type CallExpr struct {
	S      Span
	Callee string
	Args   []Expr
}

func (c *CallExpr) NodeKind() string { return "CallExpr" }
func (c *CallExpr) exprNode()        {}
func (c *CallExpr) GetSpan() Span    { return c.S }
func (c *CallExpr) String() string {
	return fmt.Sprintf("Call(%s, args=%d)", c.Callee, len(c.Args))
}

// --- Arrays v1 ---

type ArrayLiteralExpr struct {
	S        Span
	Elements []Expr
}

func (a *ArrayLiteralExpr) NodeKind() string { return "ArrayLiteralExpr" }
func (a *ArrayLiteralExpr) exprNode()        {}
func (a *ArrayLiteralExpr) GetSpan() Span    { return a.S }
func (a *ArrayLiteralExpr) String() string {
	if len(a.Elements) == 0 {
		return "Array([])"
	}
	parts := make([]string, 0, len(a.Elements))
	for _, e := range a.Elements {
		parts = append(parts, e.String())
	}
	return fmt.Sprintf("Array([%s])", strings.Join(parts, ", "))
}

type IndexExpr struct {
	S     Span
	Left  Expr
	Index Expr
}

func (x *IndexExpr) NodeKind() string { return "IndexExpr" }
func (x *IndexExpr) exprNode()        {}
func (x *IndexExpr) GetSpan() Span    { return x.S }
func (x *IndexExpr) String() string {
	return fmt.Sprintf("Index(%s, %s)", x.Left.String(), x.Index.String())
}
