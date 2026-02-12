package ast

import "fmt"

type ImportStmt struct {
	S    Span
	Path string
}

func (i *ImportStmt) NodeKind() string { return "ImportStmt" }
func (i *ImportStmt) stmtNode()        {}
func (i *ImportStmt) GetSpan() Span    { return i.S }
func (i *ImportStmt) String() string   { return fmt.Sprintf("Import(%q)", i.Path) }
