package ast

import "fmt"

type MapEntry struct {
	Key   string
	Value Expr
}

type MapLiteralExpr struct {
	S       Span
	Entries []MapEntry
}

func (m *MapLiteralExpr) NodeKind() string { return "MapLiteralExpr" }
func (m *MapLiteralExpr) exprNode()        {}
func (m *MapLiteralExpr) GetSpan() Span    { return m.S }
func (m *MapLiteralExpr) String() string {
	return fmt.Sprintf("MapLiteral(%d entries)", len(m.Entries))
}
