package lexer

import (
	"strings"
	"unicode"
)

type Lexer struct {
	input []rune
	pos   int
	ch    rune

	line int
	col  int
}

func New(input string) *Lexer {
	l := &Lexer{
		input: []rune(input),
		pos:   -1,
		ch:    0,
		line:  1,
		col:   0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	l.pos++
	if l.pos >= len(l.input) {
		l.ch = 0
		return
	}
	l.ch = l.input[l.pos]
	if l.ch == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}
}

func (l *Lexer) peekChar() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) NextToken() Token {
	// Skip spaces/tabs (but not newlines)
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}

	tok := Token{Line: l.line, Col: l.col}

	switch l.ch {
	case 0:
		tok.Type = EOF
		return tok

	case '\n':
		tok.Type = NEWLINE
		tok.Lexeme = "\n"
		l.readChar()
		return tok

	case '#':
		// IMPORTANT:
		// - If '#' followed by a digit => file-handle token HASH, then the NUMBER token follows.
		// - Otherwise '#' begins a comment to end-of-line.
		next := l.peekChar()
		if next != 0 && unicode.IsDigit(next) {
			tok.Type = HASH
			tok.Lexeme = "#"
			l.readChar()
			return tok
		}

		// comment: consume until newline or EOF
		for l.ch != 0 && l.ch != '\n' {
			l.readChar()
		}
		// do not consume newline here; let it be tokenized next call
		return l.NextToken()

	case '=':
		if l.peekChar() == '=' {
			tok.Type = EQ
			tok.Lexeme = "=="
			l.readChar()
			l.readChar()
			return tok
		}
		tok.Type = ASSIGN
		tok.Lexeme = "="
		l.readChar()
		return tok

	case '!':
		if l.peekChar() == '=' {
			tok.Type = NEQ
			tok.Lexeme = "!="
			l.readChar()
			l.readChar()
			return tok
		}
		tok.Type = ILLEGAL
		tok.Lexeme = "!"
		l.readChar()
		return tok

	case '<':
		if l.peekChar() == '=' {
			tok.Type = LTE
			tok.Lexeme = "<="
			l.readChar()
			l.readChar()
			return tok
		}
		tok.Type = LT
		tok.Lexeme = "<"
		l.readChar()
		return tok

	case '>':
		if l.peekChar() == '=' {
			tok.Type = GTE
			tok.Lexeme = ">="
			l.readChar()
			l.readChar()
			return tok
		}
		tok.Type = GT
		tok.Lexeme = ">"
		l.readChar()
		return tok

	case '+':
		tok.Type = PLUS
		tok.Lexeme = "+"
		l.readChar()
		return tok

	case '-':
		tok.Type = MINUS
		tok.Lexeme = "-"
		l.readChar()
		return tok

	case '*':
		tok.Type = STAR
		tok.Lexeme = "*"
		l.readChar()
		return tok

	case '/':
		tok.Type = SLASH
		tok.Lexeme = "/"
		l.readChar()
		return tok

	case '(':
		tok.Type = LPAREN
		tok.Lexeme = "("
		l.readChar()
		return tok

	case ')':
		tok.Type = RPAREN
		tok.Lexeme = ")"
		l.readChar()
		return tok

	case '[':
		tok.Type = LBRACKET
		tok.Lexeme = "["
		l.readChar()
		return tok

	case ']':
		tok.Type = RBRACKET
		tok.Lexeme = "]"
		l.readChar()
		return tok

	case '{':
		tok.Type = LBRACE
		tok.Lexeme = "{"
		l.readChar()
		return tok

	case '}':
		tok.Type = RBRACE
		tok.Lexeme = "}"
		l.readChar()
		return tok

	case ':':
		tok.Type = COLON
		tok.Lexeme = ":"
		l.readChar()
		return tok

	case ',':
		tok.Type = COMMA
		tok.Lexeme = ","
		l.readChar()
		return tok

	case '"':
		tok.Type = STRING
		tok.Lexeme = l.readString()
		return tok

	default:
		if isLetter(l.ch) {
			lit := l.readIdent()
			tok.Type = LookupIdent(lit)
			tok.Lexeme = lit
			return tok
		}
		if isDigit(l.ch) {
			num := l.readNumber()
			tok.Type = NUMBER
			tok.Lexeme = num
			return tok
		}

		tok.Type = ILLEGAL
		tok.Lexeme = string(l.ch)
		l.readChar()
		return tok
	}
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}

func (l *Lexer) readIdent() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readNumber() string {
	start := l.pos
	dotSeen := false
	for isDigit(l.ch) || (!dotSeen && l.ch == '.') {
		if l.ch == '.' {
			dotSeen = true
		}
		l.readChar()
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readString() string {
	// current char is '"'
	l.readChar()

	var b strings.Builder
	for l.ch != 0 && l.ch != '"' {
		if l.ch == '\\' {
			switch l.peekChar() {
			case 'n':
				b.WriteRune('\n')
				l.readChar()
				l.readChar()
				continue
			case 't':
				b.WriteRune('\t')
				l.readChar()
				l.readChar()
				continue
			case '"':
				b.WriteRune('"')
				l.readChar()
				l.readChar()
				continue
			case '\\':
				b.WriteRune('\\')
				l.readChar()
				l.readChar()
				continue
			default:
				// unknown escape; keep the slash literally
				b.WriteRune('\\')
				l.readChar()
				continue
			}
		}
		b.WriteRune(l.ch)
		l.readChar()
	}

	// consume closing quote if present
	if l.ch == '"' {
		l.readChar()
	}

	return b.String()
}
