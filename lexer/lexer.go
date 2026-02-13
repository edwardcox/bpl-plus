package lexer

import (
	"strings"
	"unicode"
)

type Lexer struct {
	src  []rune
	pos  int
	line int
	col  int
}

func New(input string) *Lexer {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	return &Lexer{
		src:  []rune(input),
		pos:  0,
		line: 1,
		col:  1,
	}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespaceExceptNewline()

	if l.atEnd() {
		return Token{Type: EOF, Line: l.line, Col: l.col}
	}

	ch := l.peek()

	// NEWLINE
	if ch == '\n' {
		tok := Token{Type: NEWLINE, Lexeme: "\n", Line: l.line, Col: l.col}
		l.advance() // advance updates line/col
		return tok
	}

	// Comments: # ... to end of line
	if ch == '#' {
		l.skipComment()
		return l.NextToken()
	}

	// Ident / keyword
	if isIdentStart(ch) {
		startLine, startCol := l.line, l.col
		ident := l.readIdent()
		tt := LookupIdent(ident)
		return Token{Type: tt, Lexeme: ident, Line: startLine, Col: startCol}
	}

	// Number
	if unicode.IsDigit(ch) {
		startLine, startCol := l.line, l.col
		num := l.readNumber()
		return Token{Type: NUMBER, Lexeme: num, Line: startLine, Col: startCol}
	}

	// String
	if ch == '"' {
		startLine, startCol := l.line, l.col
		s, ok := l.readString()
		if !ok {
			return Token{Type: ILLEGAL, Lexeme: "unterminated string", Line: startLine, Col: startCol}
		}
		return Token{Type: STRING, Lexeme: s, Line: startLine, Col: startCol}
	}

	// Two-char operators
	if ch == '=' && l.peek2() == '=' {
		return l.make2(EQ, "==")
	}
	if ch == '!' && l.peek2() == '=' {
		return l.make2(NEQ, "!=")
	}
	if ch == '<' && l.peek2() == '=' {
		return l.make2(LTE, "<=")
	}
	if ch == '>' && l.peek2() == '=' {
		return l.make2(GTE, ">=")
	}

	// Single-char tokens
	startLine, startCol := l.line, l.col
	switch ch {
	case '=':
		l.advance()
		return Token{Type: ASSIGN, Lexeme: "=", Line: startLine, Col: startCol}
	case '+':
		l.advance()
		return Token{Type: PLUS, Lexeme: "+", Line: startLine, Col: startCol}
	case '-':
		l.advance()
		return Token{Type: MINUS, Lexeme: "-", Line: startLine, Col: startCol}
	case '*':
		l.advance()
		return Token{Type: STAR, Lexeme: "*", Line: startLine, Col: startCol}
	case '/':
		l.advance()
		return Token{Type: SLASH, Lexeme: "/", Line: startLine, Col: startCol}
	case '(':
		l.advance()
		return Token{Type: LPAREN, Lexeme: "(", Line: startLine, Col: startCol}
	case ')':
		l.advance()
		return Token{Type: RPAREN, Lexeme: ")", Line: startLine, Col: startCol}
	case '[':
		l.advance()
		return Token{Type: LBRACKET, Lexeme: "[", Line: startLine, Col: startCol}
	case ']':
		l.advance()
		return Token{Type: RBRACKET, Lexeme: "]", Line: startLine, Col: startCol}
	case '{':
		l.advance()
		return Token{Type: LBRACE, Lexeme: "{", Line: startLine, Col: startCol}
	case '}':
		l.advance()
		return Token{Type: RBRACE, Lexeme: "}", Line: startLine, Col: startCol}
	case ':':
		l.advance()
		return Token{Type: COLON, Lexeme: ":", Line: startLine, Col: startCol}
	case ',':
		l.advance()
		return Token{Type: COMMA, Lexeme: ",", Line: startLine, Col: startCol}
	case '<':
		l.advance()
		return Token{Type: LT, Lexeme: "<", Line: startLine, Col: startCol}
	case '>':
		l.advance()
		return Token{Type: GT, Lexeme: ">", Line: startLine, Col: startCol}
	default:
		l.advance()
		return Token{Type: ILLEGAL, Lexeme: string(ch), Line: startLine, Col: startCol}
	}
}

func (l *Lexer) make2(t TokenType, lex string) Token {
	startLine, startCol := l.line, l.col
	l.advance()
	l.advance()
	return Token{Type: t, Lexeme: lex, Line: startLine, Col: startCol}
}

func (l *Lexer) skipWhitespaceExceptNewline() {
	for !l.atEnd() {
		ch := l.peek()
		if ch == ' ' || ch == '\t' {
			l.advance()
			continue
		}
		break
	}
}

func (l *Lexer) skipComment() {
	for !l.atEnd() && l.peek() != '\n' {
		l.advance()
	}
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (l *Lexer) readIdent() string {
	var b strings.Builder
	for !l.atEnd() && isIdentPart(l.peek()) {
		b.WriteRune(l.peek())
		l.advance()
	}
	return b.String()
}

func (l *Lexer) readNumber() string {
	var b strings.Builder
	for !l.atEnd() && unicode.IsDigit(l.peek()) {
		b.WriteRune(l.peek())
		l.advance()
	}
	// optional decimal part
	if !l.atEnd() && l.peek() == '.' && l.pos+1 < len(l.src) && unicode.IsDigit(l.src[l.pos+1]) {
		b.WriteRune('.')
		l.advance()
		for !l.atEnd() && unicode.IsDigit(l.peek()) {
			b.WriteRune(l.peek())
			l.advance()
		}
	}
	return b.String()
}

func (l *Lexer) readString() (string, bool) {
	// consume opening "
	l.advance()

	var b strings.Builder
	for !l.atEnd() {
		ch := l.peek()

		// end
		if ch == '"' {
			l.advance()
			return b.String(), true
		}

		// escape
		if ch == '\\' {
			l.advance()
			if l.atEnd() {
				return "", false
			}
			esc := l.peek()
			switch esc {
			case 'n':
				b.WriteRune('\n')
			case 'r':
				b.WriteRune('\r')
			case 't':
				b.WriteRune('\t')
			case '"':
				b.WriteRune('"')
			case '\\':
				b.WriteRune('\\')
			default:
				// unknown escape: keep literal
				b.WriteRune(esc)
			}
			l.advance()
			continue
		}

		// raw char
		b.WriteRune(ch)
		l.advance()
	}

	return "", false
}

func (l *Lexer) atEnd() bool { return l.pos >= len(l.src) }

func (l *Lexer) peek() rune { return l.src[l.pos] }

func (l *Lexer) peek2() rune {
	if l.pos+1 >= len(l.src) {
		return 0
	}
	return l.src[l.pos+1]
}

func (l *Lexer) advance() {
	if l.atEnd() {
		return
	}

	ch := l.src[l.pos]
	l.pos++

	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
}
