package ast

type Span struct {
	Line int
	Col  int
}

type HasSpan interface {
	GetSpan() Span
}

func SpanOf(n any) (Span, bool) {
	if n == nil {
		return Span{}, false
	}
	hs, ok := n.(HasSpan)
	if !ok {
		return Span{}, false
	}
	return hs.GetSpan(), true
}
