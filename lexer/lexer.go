package lexer

type Lexer struct {
	input []rune
	pos   int
	line  int
	col   int
}

func New(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		line:  1,
		col:   1,
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) advance() rune {
	ch := l.peek()
	if ch == 0 {
		return 0
	}
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) NextToken() Token {
	// skip spaces/tabs/carriage returns (not newline)
	for {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
			continue
		}
		break
	}

	startLine := l.line
	startCol := l.col
	ch := l.peek()

	if ch == 0 {
		return Token{Type: EOF, Line: startLine, Col: startCol}
	}

	if ch == '\n' {
		l.advance()
		return Token{Type: NEWLINE, Lexeme: "\n", Line: startLine, Col: startCol}
	}

	// comment starting with '
	if ch == '\'' {
		for ch != 0 && ch != '\n' {
			ch = l.advance()
		}
		return l.NextToken()
	}

	// identifiers/keywords
	if isAlpha(ch) || ch == '_' {
		lex := ""
		for isAlphaNum(l.peek()) || l.peek() == '_' {
			lex += string(l.advance())
		}
		tt := LookupIdent(lex)
		return Token{Type: tt, Lexeme: lex, Line: startLine, Col: startCol}
	}

	// numbers (int/float)
	if isDigit(ch) {
		lex := ""
		dotSeen := false
		for {
			c := l.peek()
			if isDigit(c) {
				lex += string(l.advance())
				continue
			}
			if c == '.' && !dotSeen {
				dotSeen = true
				lex += string(l.advance())
				continue
			}
			break
		}
		return Token{Type: NUMBER, Lexeme: lex, Line: startLine, Col: startCol}
	}

	// strings "..."
	if ch == '"' {
		l.advance()
		lex := ""
		for {
			c := l.peek()
			if c == 0 || c == '\n' {
				return Token{Type: ILLEGAL, Lexeme: "Unterminated string", Line: startLine, Col: startCol}
			}
			if c == '"' {
				l.advance()
				break
			}
			if c == '\\' {
				l.advance()
				esc := l.peek()
				if esc == 0 {
					return Token{Type: ILLEGAL, Lexeme: "Bad escape", Line: startLine, Col: startCol}
				}
				switch esc {
				case 'n':
					l.advance()
					lex += "\n"
				case 't':
					l.advance()
					lex += "\t"
				case '"':
					l.advance()
					lex += `"`
				case '\\':
					l.advance()
					lex += `\`
				default:
					lex += string(esc)
					l.advance()
				}
				continue
			}
			lex += string(l.advance())
		}
		return Token{Type: STRING, Lexeme: lex, Line: startLine, Col: startCol}
	}

	// two-char operators / assignment
	if ch == '=' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: EQ, Lexeme: "==", Line: startLine, Col: startCol}
		}
		return Token{Type: ASSIGN, Lexeme: "=", Line: startLine, Col: startCol}
	}
	if ch == '!' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: NEQ, Lexeme: "!=", Line: startLine, Col: startCol}
		}
		return Token{Type: ILLEGAL, Lexeme: "!", Line: startLine, Col: startCol}
	}
	if ch == '<' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: LTE, Lexeme: "<=", Line: startLine, Col: startCol}
		}
		return Token{Type: LT, Lexeme: "<", Line: startLine, Col: startCol}
	}
	if ch == '>' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: GTE, Lexeme: ">=", Line: startLine, Col: startCol}
		}
		return Token{Type: GT, Lexeme: ">", Line: startLine, Col: startCol}
	}

	// single-char tokens
	switch ch {
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
	case ',':
		l.advance()
		return Token{Type: COMMA, Lexeme: ",", Line: startLine, Col: startCol}
	}

	l.advance()
	return Token{Type: ILLEGAL, Lexeme: string(ch), Line: startLine, Col: startCol}
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlphaNum(r rune) bool {
	return isAlpha(r) || isDigit(r)
}
