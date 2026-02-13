package parser

import (
	"fmt"

	"bpl-plus/ast"
	"bpl-plus/lexer"
)

type Parser struct {
	lx   *lexer.Lexer
	cur  lexer.Token
	peek lexer.Token
}

func New(lx *lexer.Lexer) *Parser {
	p := &Parser{lx: lx}
	p.cur = lx.NextToken()
	p.peek = lx.NextToken()
	return p
}

func (p *Parser) next() {
	p.cur = p.peek
	p.peek = p.lx.NextToken()
}

func sp(tok lexer.Token) ast.Span { return ast.Span{Line: tok.Line, Col: tok.Col} }

func (p *Parser) ParseProgram() ([]ast.Stmt, error) {
	stmts := []ast.Stmt{}
	for p.cur.Type != lexer.EOF {
		if p.cur.Type == lexer.NEWLINE {
			p.next()
			continue
		}
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		for p.cur.Type == lexer.NEWLINE {
			p.next()
		}
	}
	return stmts, nil
}

func (p *Parser) parseStmt() (ast.Stmt, error) {
	switch p.cur.Type {
	case lexer.PRINT:
		return p.parsePrint()
	case lexer.IF:
		return p.parseIf()
	case lexer.WHILE:
		return p.parseWhile()
	case lexer.FOR:
		return p.parseFor()
	case lexer.FUNCTION:
		return p.parseFunctionDecl()
	case lexer.RETURN:
		return p.parseReturn()
	case lexer.IMPORT:
		return p.parseImport()
	default:
		// index assignment: a[i] = ...
		if p.cur.Type == lexer.IDENT && p.peek.Type == lexer.LBRACKET {
			return p.parseIndexAssign()
		}
		// normal assignment: a = ...
		if p.cur.Type == lexer.IDENT && p.peek.Type == lexer.ASSIGN {
			return p.parseAssign()
		}
		// expression statement: push(a, 1)
		if p.cur.Type == lexer.IDENT && p.peek.Type == lexer.LPAREN {
			return p.parseExprStmt()
		}
		return nil, p.errAt(p.cur, "Expected a statement")
	}
}

func (p *Parser) parsePrint() (ast.Stmt, error) {
	printTok := p.cur
	p.next()
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ast.PrintStmt{S: sp(printTok), Value: expr}, nil
}

func (p *Parser) parseAssign() (ast.Stmt, error) {
	nameTok := p.cur
	p.next() // move to '='
	p.next() // move to expr
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ast.AssignStmt{S: sp(nameTok), Name: nameTok.Lexeme, Value: expr}, nil
}

// indexAssign = IDENT "[" expr "]" "=" expr
func (p *Parser) parseIndexAssign() (ast.Stmt, error) {
	nameTok := p.cur
	name := nameTok.Lexeme

	p.next() // to '['
	if p.cur.Type != lexer.LBRACKET {
		return nil, p.errAt(p.cur, "Expected '[' after identifier")
	}
	lbTok := p.cur

	p.next()
	indexExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if p.cur.Type != lexer.RBRACKET {
		return nil, p.errAt(p.cur, "Expected ']' after index expression")
	}
	p.next()

	if p.cur.Type != lexer.ASSIGN {
		return nil, p.errAt(p.cur, "Expected '=' after index expression")
	}
	p.next()

	valExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &ast.IndexAssignStmt{S: sp(lbTok), Name: name, Index: indexExpr, Value: valExpr}, nil
}

// exprStmt = expr   (we only allow this in practice for calls right now)
func (p *Parser) parseExprStmt() (ast.Stmt, error) {
	startTok := p.cur
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ast.ExprStmt{S: sp(startTok), Expr: expr}, nil
}

func (p *Parser) parseReturn() (ast.Stmt, error) {
	retTok := p.cur
	p.next()
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ast.ReturnStmt{S: sp(retTok), Value: expr}, nil
}

// importStmt = "import" STRING
func (p *Parser) parseImport() (ast.Stmt, error) {
	imTok := p.cur
	p.next()
	if p.cur.Type != lexer.STRING {
		return nil, p.errAt(p.cur, "Expected string path after 'import'")
	}
	pathTok := p.cur
	p.next()
	return &ast.ImportStmt{S: sp(imTok), Path: pathTok.Lexeme}, nil
}

func (p *Parser) parseFunctionDecl() (ast.Stmt, error) {
	p.next()
	if p.cur.Type != lexer.IDENT {
		return nil, p.errAt(p.cur, "Expected function name after 'function'")
	}
	nameTok := p.cur
	name := nameTok.Lexeme

	p.next()
	if p.cur.Type != lexer.LPAREN {
		return nil, p.errAt(p.cur, "Expected '(' after function name")
	}

	params := []string{}
	p.next()
	if p.cur.Type != lexer.RPAREN {
		for {
			if p.cur.Type != lexer.IDENT {
				return nil, p.errAt(p.cur, "Expected parameter name")
			}
			params = append(params, p.cur.Lexeme)

			p.next()
			if p.cur.Type == lexer.COMMA {
				p.next()
				continue
			}
			if p.cur.Type == lexer.RPAREN {
				break
			}
			return nil, p.errAt(p.cur, "Expected ',' or ')' in parameter list")
		}
	}

	if p.cur.Type != lexer.RPAREN {
		return nil, p.errAt(p.cur, "Expected ')' after parameters")
	}
	p.next()

	if p.cur.Type != lexer.NEWLINE {
		return nil, p.errAt(p.cur, "Expected NEWLINE after function header")
	}
	for p.cur.Type == lexer.NEWLINE {
		p.next()
	}

	body, err := p.parseBlockUntil(lexer.END)
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.END {
		return nil, p.errAt(p.cur, "Expected 'end' to close function")
	}
	p.next()

	return &ast.FunctionDecl{S: sp(nameTok), Name: name, Params: params, Body: body}, nil
}

func (p *Parser) parseIf() (ast.Stmt, error) {
	ifTok := p.cur
	p.next()
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.NEWLINE {
		return nil, p.errAt(p.cur, "Expected NEWLINE after if condition")
	}
	for p.cur.Type == lexer.NEWLINE {
		p.next()
	}

	thenBlock, err := p.parseBlockUntil(lexer.ELSE, lexer.END)
	if err != nil {
		return nil, err
	}

	elseBlock := []ast.Stmt{}
	if p.cur.Type == lexer.ELSE {
		p.next()
		if p.cur.Type != lexer.NEWLINE {
			return nil, p.errAt(p.cur, "Expected NEWLINE after else")
		}
		for p.cur.Type == lexer.NEWLINE {
			p.next()
		}
		elseBlock, err = p.parseBlockUntil(lexer.END)
		if err != nil {
			return nil, err
		}
	}

	if p.cur.Type != lexer.END {
		return nil, p.errAt(p.cur, "Expected 'end' to close if")
	}
	p.next()

	return &ast.IfStmt{S: sp(ifTok), Condition: cond, Then: thenBlock, Else: elseBlock}, nil
}

func (p *Parser) parseWhile() (ast.Stmt, error) {
	wTok := p.cur
	p.next()
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.NEWLINE {
		return nil, p.errAt(p.cur, "Expected NEWLINE after while condition")
	}
	for p.cur.Type == lexer.NEWLINE {
		p.next()
	}

	body, err := p.parseBlockUntil(lexer.END)
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.END {
		return nil, p.errAt(p.cur, "Expected 'end' to close while")
	}
	p.next()

	return &ast.WhileStmt{S: sp(wTok), Condition: cond, Body: body}, nil
}

func (p *Parser) parseFor() (ast.Stmt, error) {
	p.next()
	if p.cur.Type != lexer.IDENT {
		return nil, p.errAt(p.cur, "Expected loop variable after 'for'")
	}
	varNameTok := p.cur
	varName := varNameTok.Lexeme

	p.next()
	if p.cur.Type != lexer.ASSIGN {
		return nil, p.errAt(p.cur, "Expected '=' after loop variable")
	}

	p.next()
	startExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.TO {
		return nil, p.errAt(p.cur, "Expected 'to' in for loop")
	}

	p.next()
	endExpr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	var stepExpr ast.Expr = nil
	if p.cur.Type == lexer.STEP {
		p.next()
		stepExpr, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
	}

	if p.cur.Type != lexer.NEWLINE {
		return nil, p.errAt(p.cur, "Expected NEWLINE after for header")
	}
	for p.cur.Type == lexer.NEWLINE {
		p.next()
	}

	body, err := p.parseBlockUntil(lexer.END)
	if err != nil {
		return nil, err
	}
	if p.cur.Type != lexer.END {
		return nil, p.errAt(p.cur, "Expected 'end' to close for")
	}
	p.next()

	return &ast.ForStmt{S: sp(varNameTok), Var: varName, Start: startExpr, End: endExpr, Step: stepExpr, Body: body}, nil
}

func (p *Parser) parseBlockUntil(terminators ...lexer.TokenType) ([]ast.Stmt, error) {
	block := []ast.Stmt{}
	for p.cur.Type != lexer.EOF && !p.isOneOf(p.cur.Type, terminators...) {
		if p.cur.Type == lexer.NEWLINE {
			p.next()
			continue
		}
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		block = append(block, stmt)
		for p.cur.Type == lexer.NEWLINE {
			p.next()
		}
	}
	return block, nil
}

func (p *Parser) isOneOf(t lexer.TokenType, list ...lexer.TokenType) bool {
	for _, x := range list {
		if t == x {
			return true
		}
	}
	return false
}

// expr = or
func (p *Parser) parseExpr() (ast.Expr, error) { return p.parseOr() }

// or = and ( "or" and )*
func (p *Parser) parseOr() (ast.Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.OR {
		opTok := p.cur
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{S: sp(opTok), Left: left, Op: "or", Right: right}
	}
	return left, nil
}

// and = comparison ( "and" comparison )*
func (p *Parser) parseAnd() (ast.Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.AND {
		opTok := p.cur
		p.next()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{S: sp(opTok), Left: left, Op: "and", Right: right}
	}
	return left, nil
}

// comparison = addsub ( (==|!=|<|>|<=|>=) addsub )?
func (p *Parser) parseComparison() (ast.Expr, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	if isCompareTok(p.cur.Type) {
		opTok := p.cur
		op := p.cur.Lexeme
		p.next()
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpr{S: sp(opTok), Left: left, Op: op, Right: right}, nil
	}
	return left, nil
}

func isCompareTok(t lexer.TokenType) bool {
	return t == lexer.EQ || t == lexer.NEQ || t == lexer.LT || t == lexer.GT || t == lexer.LTE || t == lexer.GTE
}

func (p *Parser) parseAddSub() (ast.Expr, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.PLUS || p.cur.Type == lexer.MINUS {
		opTok := p.cur
		op := p.cur.Lexeme
		p.next()
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{S: sp(opTok), Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseMulDiv() (ast.Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.cur.Type == lexer.STAR || p.cur.Type == lexer.SLASH {
		opTok := p.cur
		op := p.cur.Lexeme
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{S: sp(opTok), Left: left, Op: op, Right: right}
	}
	return left, nil
}

// unary = ("not") unary | postfix
func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.cur.Type == lexer.NOT {
		opTok := p.cur
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{S: sp(opTok), Op: "not", Right: right}, nil
	}
	return p.parsePostfix()
}

// postfix = primary ( "[" expr "]" )*
func (p *Parser) parsePostfix() (ast.Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.cur.Type == lexer.LBRACKET {
		brTok := p.cur
		p.next()

		indexExpr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		if p.cur.Type != lexer.RBRACKET {
			return nil, p.errAt(p.cur, "Expected ']' after index expression")
		}
		p.next()

		left = &ast.IndexExpr{S: sp(brTok), Left: left, Index: indexExpr}
	}

	return left, nil
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	switch p.cur.Type {
	case lexer.STRING:
		tok := p.cur
		expr := &ast.StringLiteral{S: sp(tok), Value: tok.Lexeme}
		p.next()
		return expr, nil

	case lexer.NUMBER:
		tok := p.cur
		expr := &ast.NumberLiteral{S: sp(tok), Lexeme: tok.Lexeme}
		p.next()
		return expr, nil

	case lexer.TRUE:
		tok := p.cur
		p.next()
		return &ast.BoolLiteral{S: sp(tok), Value: true}, nil

	case lexer.FALSE:
		tok := p.cur
		p.next()
		return &ast.BoolLiteral{S: sp(tok), Value: false}, nil

	case lexer.IDENT:
		nameTok := p.cur
		name := p.cur.Lexeme
		p.next()

		if p.cur.Type == lexer.LPAREN {
			args := []ast.Expr{}
			p.next()
			if p.cur.Type != lexer.RPAREN {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)

					if p.cur.Type == lexer.COMMA {
						p.next()
						continue
					}
					if p.cur.Type == lexer.RPAREN {
						break
					}
					return nil, p.errAt(p.cur, "Expected ',' or ')' in call arguments")
				}
			}
			if p.cur.Type != lexer.RPAREN {
				return nil, p.errAt(p.cur, "Expected ')' after call arguments")
			}
			p.next()
			return &ast.CallExpr{S: sp(nameTok), Callee: name, Args: args}, nil
		}

		return &ast.Identifier{S: sp(nameTok), Name: name}, nil

	case lexer.LPAREN:
		p.next()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.cur.Type != lexer.RPAREN {
			return nil, p.errAt(p.cur, "Expected ')'")
		}
		p.next()
		return expr, nil

	case lexer.LBRACKET:
		return p.parseArrayLiteral()

	// ✅ Maps Step C: map literal primary
	case lexer.LBRACE:
		return p.parseMapLiteral()

	default:
		return nil, p.errAt(p.cur, "Expected an expression")
	}
}

// arrayLiteral = "[" [ expr ("," expr)* ] "]"
func (p *Parser) parseArrayLiteral() (ast.Expr, error) {
	lbTok := p.cur
	p.next()

	elems := []ast.Expr{}

	if p.cur.Type == lexer.RBRACKET {
		p.next()
		return &ast.ArrayLiteralExpr{S: sp(lbTok), Elements: elems}, nil
	}

	for {
		elem, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)

		if p.cur.Type == lexer.COMMA {
			p.next()
			continue
		}
		if p.cur.Type == lexer.RBRACKET {
			p.next()
			break
		}
		return nil, p.errAt(p.cur, "Expected ',' or ']' in array literal")
	}

	return &ast.ArrayLiteralExpr{S: sp(lbTok), Elements: elems}, nil
}

// mapLiteral = "{" [ string ":" expr ("," string ":" expr)* ] "}"
// Keys are STRING tokens (so you write: {"a": 1, "b": 2})
func (p *Parser) parseMapLiteral() (ast.Expr, error) {
	lbTok := p.cur // '{'
	p.next()

	entries := []ast.MapEntry{}

	// empty map
	if p.cur.Type == lexer.RBRACE {
		p.next()
		return &ast.MapLiteralExpr{S: sp(lbTok), Entries: entries}, nil
	}

	for {
		if p.cur.Type != lexer.STRING {
			return nil, p.errAt(p.cur, "Expected string key in map literal")
		}
		keyTok := p.cur
		key := keyTok.Lexeme
		p.next()

		if p.cur.Type != lexer.COLON {
			return nil, p.errAt(p.cur, "Expected ':' after map key")
		}
		p.next()

		val, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		entries = append(entries, ast.MapEntry{Key: key, Value: val})

		if p.cur.Type == lexer.COMMA {
			p.next()
			continue
		}
		if p.cur.Type == lexer.RBRACE {
			p.next()
			break
		}
		return nil, p.errAt(p.cur, "Expected ',' or '}' in map literal")
	}

	return &ast.MapLiteralExpr{S: sp(lbTok), Entries: entries}, nil
}

func (p *Parser) errAt(tok lexer.Token, msg string) error {
	if tok.Type == lexer.EOF {
		return fmt.Errorf("%s at end of file", msg)
	}
	return fmt.Errorf("%s at %d:%d (got %s)", msg, tok.Line, tok.Col, tok.Type)
}
