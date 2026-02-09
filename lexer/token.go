package lexer

import "fmt"

type TokenType string

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"
	NEWLINE TokenType = "NEWLINE"

	IDENT  TokenType = "IDENT"
	NUMBER TokenType = "NUMBER"
	STRING TokenType = "STRING"

	PRINT    TokenType = "PRINT"
	IF       TokenType = "IF"
	ELSE     TokenType = "ELSE"
	END      TokenType = "END"
	WHILE    TokenType = "WHILE"
	FOR      TokenType = "FOR"
	TO       TokenType = "TO"
	STEP     TokenType = "STEP"
	FUNCTION TokenType = "FUNCTION"
	RETURN   TokenType = "RETURN"
	TRUE     TokenType = "TRUE"
	FALSE    TokenType = "FALSE"

	// NEW: foreach sugar tokens
	EACH TokenType = "EACH"
	IN   TokenType = "IN"

	AND TokenType = "AND"
	OR  TokenType = "OR"
	NOT TokenType = "NOT"

	ASSIGN TokenType = "ASSIGN"
	PLUS   TokenType = "PLUS"
	MINUS  TokenType = "MINUS"
	STAR   TokenType = "STAR"
	SLASH  TokenType = "SLASH"

	LPAREN   TokenType = "LPAREN"
	RPAREN   TokenType = "RPAREN"
	LBRACKET TokenType = "LBRACKET"
	RBRACKET TokenType = "RBRACKET"

	COMMA TokenType = "COMMA"

	EQ  TokenType = "EQ"
	NEQ TokenType = "NEQ"
	LT  TokenType = "LT"
	GT  TokenType = "GT"
	LTE TokenType = "LTE"
	GTE TokenType = "GTE"
)

type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Col    int
}

func (t Token) String() string {
	switch t.Type {
	case STRING:
		return fmt.Sprintf("%s(%q) @ %d:%d", t.Type, t.Lexeme, t.Line, t.Col)
	case IDENT, NUMBER:
		return fmt.Sprintf("%s(%s) @ %d:%d", t.Type, t.Lexeme, t.Line, t.Col)
	default:
		return fmt.Sprintf("%s @ %d:%d", t.Type, t.Line, t.Col)
	}
}

func LookupIdent(ident string) TokenType {
	switch ident {
	case "print", "PRINT", "Print":
		return PRINT
	case "if", "IF", "If":
		return IF
	case "else", "ELSE", "Else":
		return ELSE
	case "end", "END", "End":
		return END
	case "while", "WHILE", "While":
		return WHILE
	case "for", "FOR", "For":
		return FOR
	case "to", "TO", "To":
		return TO
	case "step", "STEP", "Step":
		return STEP
	case "function", "FUNCTION", "Function":
		return FUNCTION
	case "return", "RETURN", "Return":
		return RETURN
	case "true", "TRUE", "True":
		return TRUE
	case "false", "FALSE", "False":
		return FALSE

	// NEW: foreach keywords
	case "each", "EACH", "Each":
		return EACH
	case "in", "IN", "In":
		return IN

	case "and", "AND", "And":
		return AND
	case "or", "OR", "Or":
		return OR
	case "not", "NOT", "Not":
		return NOT

	default:
		return IDENT
	}
}
