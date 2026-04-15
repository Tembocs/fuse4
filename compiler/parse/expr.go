package parse

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/lex"
)

// Precedence levels for Pratt parsing.
type precedence int

const (
	precNone    precedence = iota
	precAssign             // = += -= etc.
	precOr                 // ||
	precAnd                // &&
	precEq                 // == !=
	precCmp                // < > <= >=
	precBitOr              // |
	precBitXor             // ^
	precBitAnd             // &
	precShift              // << >>
	precAdd                // + -
	precMul                // * / %
)

func infixPrec(kind lex.TokenKind) precedence {
	switch kind {
	case lex.Eq, lex.PlusEq, lex.MinusEq, lex.StarEq, lex.SlashEq,
		lex.PercentEq, lex.AmpEq, lex.PipeEq, lex.CaretEq,
		lex.ShlEq, lex.ShrEq:
		return precAssign
	case lex.PipePipe:
		return precOr
	case lex.AmpAmp:
		return precAnd
	case lex.EqEq, lex.BangEq:
		return precEq
	case lex.Lt, lex.Gt, lex.LtEq, lex.GtEq:
		return precCmp
	case lex.Pipe:
		return precBitOr
	case lex.Caret:
		return precBitXor
	case lex.Amp:
		return precBitAnd
	case lex.Shl, lex.Shr:
		return precShift
	case lex.Plus, lex.Minus:
		return precAdd
	case lex.Star, lex.Slash, lex.Percent:
		return precMul
	}
	return precNone
}

func isAssignOp(kind lex.TokenKind) bool {
	switch kind {
	case lex.Eq, lex.PlusEq, lex.MinusEq, lex.StarEq, lex.SlashEq,
		lex.PercentEq, lex.AmpEq, lex.PipeEq, lex.CaretEq,
		lex.ShlEq, lex.ShrEq:
		return true
	}
	return false
}

// parseExpr is the main expression entry point.
func (p *Parser) parseExpr() ast.Expr {
	return p.parsePrecExpr(precAssign)
}

// parsePrecExpr parses an expression at the given minimum precedence.
func (p *Parser) parsePrecExpr(minPrec precedence) ast.Expr {
	left := p.parseUnaryExpr()

	for {
		prec := infixPrec(p.peekKind())
		if prec == precNone {
			break
		}
		// For left-assoc operators, break when prec <= minPrec.
		// For right-assoc (assignment), break when prec < minPrec.
		if isAssignOp(p.peekKind()) {
			if prec < minPrec {
				break
			}
		} else {
			if prec <= minPrec {
				break
			}
		}

		op := p.advance()

		if isAssignOp(op.Kind) {
			right := p.parsePrecExpr(precAssign) // right-assoc
			left = &ast.AssignExpr{
				Span:   spanBetween(left, right),
				Target: left,
				Op:     op,
				Value:  right,
			}
		} else {
			right := p.parsePrecExpr(prec)
			left = &ast.BinaryExpr{
				Span:  spanBetween(left, right),
				Left:  left,
				Op:    op,
				Right: right,
			}
		}
	}

	return left
}

// parseUnaryExpr parses prefix unary operators.
func (p *Parser) parseUnaryExpr() ast.Expr {
	switch p.peekKind() {
	case lex.Bang, lex.Minus, lex.Tilde:
		op := p.advance()
		operand := p.parseUnaryExpr()
		return &ast.UnaryExpr{
			Span:    spanToNode(op.Span, operand),
			Op:      op,
			Operand: operand,
		}
	case lex.KwRef, lex.KwMutref, lex.KwOwned, lex.KwMove:
		op := p.advance()
		operand := p.parseUnaryExpr()
		return &ast.UnaryExpr{
			Span:    spanToNode(op.Span, operand),
			Op:      op,
			Operand: operand,
		}
	}
	return p.parsePostfixExpr()
}

// parsePostfixExpr handles ., ?., ?, (), [].
func (p *Parser) parsePostfixExpr() ast.Expr {
	expr := p.parsePrimaryExpr()

	for {
		switch p.peekKind() {
		case lex.Dot:
			p.advance() // .
			// tuple field access (numeric) or named field
			if p.at(lex.IntLit) {
				idx := p.advance()
				expr = &ast.FieldExpr{
					Span: spanStartEnd(expr.NodeSpan(), idx.Span),
					Expr: expr,
					Name: idx.Literal,
				}
			} else {
				name := p.expect(lex.Ident)
				expr = &ast.FieldExpr{
					Span: spanStartEnd(expr.NodeSpan(), name.Span),
					Expr: expr,
					Name: name.Literal,
				}
			}

		case lex.QDot:
			p.advance() // ?.
			name := p.expect(lex.Ident)
			expr = &ast.QDotExpr{
				Span: spanStartEnd(expr.NodeSpan(), name.Span),
				Expr: expr,
				Name: name.Literal,
			}

		case lex.Question:
			tok := p.advance() // ?
			expr = &ast.QuestionExpr{
				Span: spanStartEnd(expr.NodeSpan(), tok.Span),
				Expr: expr,
			}

		case lex.LParen:
			expr = p.parseCallExpr(expr)

		case lex.LBrack:
			expr = p.parseIndexExpr(expr)

		default:
			return expr
		}
	}
}

func (p *Parser) parseCallExpr(callee ast.Expr) *ast.CallExpr {
	p.advance() // (
	var args []ast.Expr
	for !p.at(lex.RParen) && !p.at(lex.EOF) {
		args = append(args, p.parseExpr())
		if !p.at(lex.RParen) {
			p.expect(lex.Comma)
		}
	}
	end := p.expect(lex.RParen)
	return &ast.CallExpr{
		Span:   spanStartEnd(callee.NodeSpan(), end.Span),
		Callee: callee,
		Args:   args,
	}
}

func (p *Parser) parseIndexExpr(expr ast.Expr) *ast.IndexExpr {
	p.advance() // [
	first := p.parseExpr()
	// If there are commas, collect multiple expressions (type argument list).
	if p.at(lex.Comma) {
		elems := []ast.Expr{first}
		for p.at(lex.Comma) {
			p.advance()
			elems = append(elems, p.parseExpr())
		}
		end := p.expect(lex.RBrack)
		return &ast.IndexExpr{
			Span: spanStartEnd(expr.NodeSpan(), end.Span),
			Expr: expr,
			Index: &ast.TupleExpr{
				Span:  spanStartEnd(first.NodeSpan(), elems[len(elems)-1].NodeSpan()),
				Elems: elems,
			},
		}
	}
	end := p.expect(lex.RBrack)
	return &ast.IndexExpr{
		Span:  spanStartEnd(expr.NodeSpan(), end.Span),
		Expr:  expr,
		Index: first,
	}
}

// parsePrimaryExpr parses primary (atomic) expressions.
func (p *Parser) parsePrimaryExpr() ast.Expr {
	switch p.peekKind() {
	// --- literals ---
	case lex.IntLit, lex.FloatLit, lex.StringLit, lex.RawStringLit:
		tok := p.advance()
		return &ast.LiteralExpr{Span: tok.Span, Token: tok}
	case lex.KwTrue, lex.KwFalse, lex.KwNone:
		tok := p.advance()
		return &ast.LiteralExpr{Span: tok.Span, Token: tok}

	// --- Some (can be used as constructor-like) ---
	case lex.KwSome:
		tok := p.advance()
		return &ast.IdentExpr{Span: tok.Span, Name: tok.Literal}

	// --- identifiers (with struct-literal disambiguation) ---
	case lex.Ident, lex.KwSelfValue, lex.KwSelfType:
		return p.parseIdentOrStructLit()

	// --- grouping or tuple ---
	case lex.LParen:
		return p.parseParenOrTuple()

	// --- block ---
	case lex.LBrace:
		return p.parseBlock()

	// --- if ---
	case lex.KwIf:
		return p.parseIfExpr()

	// --- match ---
	case lex.KwMatch:
		return p.parseMatchExpr()

	// --- for ---
	case lex.KwFor:
		return p.parseForExpr()

	// --- while ---
	case lex.KwWhile:
		return p.parseWhileExpr()

	// --- loop ---
	case lex.KwLoop:
		return p.parseLoopExpr()

	// --- return ---
	case lex.KwReturn:
		return p.parseReturnExpr()

	// --- break ---
	case lex.KwBreak:
		return p.parseBreakExpr()

	// --- continue ---
	case lex.KwContinue:
		tok := p.advance()
		return &ast.ContinueExpr{Span: tok.Span}

	// --- spawn ---
	case lex.KwSpawn:
		return p.parseSpawnExpr()

	// --- closure: fn(...) { ... } in expression position ---
	case lex.KwFn:
		return p.parseClosureExpr()

	// --- unsafe block ---
	case lex.KwUnsafe:
		// unsafe { ... } is just a block expression preceded by unsafe keyword
		start := p.advance().Span // unsafe
		block := p.parseBlock()
		block.Span = spanStartEnd(start, block.Span)
		return block

	default:
		p.errorf(p.peek().Span, "expected expression, got %s", p.peekKind())
		tok := p.advance() // consume to avoid infinite loop
		return &ast.LiteralExpr{Span: tok.Span, Token: tok}
	}
}

// --- struct-literal disambiguation ---
// Per implementation contract: IDENT { is a struct literal only if the brace
// body is empty or begins with IDENT :.

func (p *Parser) parseIdentOrStructLit() ast.Expr {
	tok := p.advance() // ident / self / Self
	ident := &ast.IdentExpr{Span: tok.Span, Name: tok.Literal}

	if !p.at(lex.LBrace) {
		return ident
	}

	// Lookahead: is this a struct literal?
	if p.isStructLitStart() {
		return p.parseStructLitBody(tok)
	}

	return ident
}

func (p *Parser) isStructLitStart() bool {
	// We are positioned at '{'. Check what follows.
	next := p.peekAt(1) // first token inside braces
	if next.Kind == lex.RBrace {
		return true // empty struct literal: Name {}
	}
	if next.Kind == lex.Ident {
		after := p.peekAt(2)
		if after.Kind == lex.Colon {
			return true // Name { field: value, ... }
		}
	}
	return false
}

func (p *Parser) parseStructLitBody(nameTok lex.Token) *ast.StructLitExpr {
	p.advance() // {
	var fields []ast.FieldInit
	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		start := p.peek().Span
		name := p.expect(lex.Ident)
		p.expect(lex.Colon)
		value := p.parseExpr()
		fields = append(fields, ast.FieldInit{
			Span:  p.spanFrom(start),
			Name:  name.Literal,
			Value: value,
		})
		if !p.at(lex.RBrace) {
			p.expect(lex.Comma)
		}
	}
	end := p.expect(lex.RBrace)
	return &ast.StructLitExpr{
		Span:   spanStartEnd(nameTok.Span, end.Span),
		Name:   nameTok.Literal,
		Fields: fields,
	}
}

// --- parenthesized expression or tuple ---

func (p *Parser) parseParenOrTuple() ast.Expr {
	start := p.advance().Span // (

	// unit: ()
	if p.at(lex.RParen) {
		end := p.advance()
		return &ast.TupleExpr{
			Span:  spanStartEnd(start, end.Span),
			Elems: nil,
		}
	}

	first := p.parseExpr()

	// tuple: (a, b, ...)
	if p.at(lex.Comma) {
		elems := []ast.Expr{first}
		for p.at(lex.Comma) {
			p.advance()
			if p.at(lex.RParen) {
				break // trailing comma
			}
			elems = append(elems, p.parseExpr())
		}
		end := p.expect(lex.RParen)
		return &ast.TupleExpr{
			Span:  spanStartEnd(start, end.Span),
			Elems: elems,
		}
	}

	// grouping: (expr)
	p.expect(lex.RParen)
	return first
}

// --- if ---

func (p *Parser) parseIfExpr() *ast.IfExpr {
	start := p.advance().Span // if
	cond := p.parseExpr()
	then := p.parseBlock()

	var elseExpr ast.Expr
	if p.at(lex.KwElse) {
		p.advance() // else
		if p.at(lex.KwIf) {
			elseExpr = p.parseIfExpr()
		} else {
			elseExpr = p.parseBlock()
		}
	}

	return &ast.IfExpr{
		Span: p.spanFrom(start),
		Cond: cond,
		Then: then,
		Else: elseExpr,
	}
}

// --- match ---

func (p *Parser) parseMatchExpr() *ast.MatchExpr {
	start := p.advance().Span // match
	subject := p.parseExpr()

	p.expect(lex.LBrace)
	var arms []ast.MatchArm
	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		arms = append(arms, p.parseMatchArm())
		if !p.at(lex.RBrace) {
			p.expect(lex.Comma)
		}
	}
	p.expect(lex.RBrace)

	return &ast.MatchExpr{
		Span:    p.spanFrom(start),
		Subject: subject,
		Arms:    arms,
	}
}

func (p *Parser) parseMatchArm() ast.MatchArm {
	start := p.peek().Span
	pat := p.parsePattern()

	var guard ast.Expr
	if p.at(lex.KwIf) {
		p.advance()
		guard = p.parseExpr()
	}

	p.expect(lex.FatArrow)
	body := p.parseExpr()

	return ast.MatchArm{
		Span:    p.spanFrom(start),
		Pattern: pat,
		Guard:   guard,
		Body:    body,
	}
}

// --- for ---

func (p *Parser) parseForExpr() *ast.ForExpr {
	start := p.advance().Span // for
	binding := p.expect(lex.Ident)
	p.expect(lex.KwIn)
	iter := p.parseExpr()
	body := p.parseBlock()

	return &ast.ForExpr{
		Span:     p.spanFrom(start),
		Binding:  binding.Literal,
		Iterable: iter,
		Body:     body,
	}
}

// --- while ---

func (p *Parser) parseWhileExpr() *ast.WhileExpr {
	start := p.advance().Span // while
	cond := p.parseExpr()
	body := p.parseBlock()
	return &ast.WhileExpr{
		Span: p.spanFrom(start),
		Cond: cond,
		Body: body,
	}
}

// --- loop ---

func (p *Parser) parseLoopExpr() *ast.LoopExpr {
	start := p.advance().Span // loop
	body := p.parseBlock()
	return &ast.LoopExpr{
		Span: p.spanFrom(start),
		Body: body,
	}
}

// --- return ---

func (p *Parser) parseReturnExpr() *ast.ReturnExpr {
	start := p.advance().Span // return
	var value ast.Expr
	if !p.at(lex.Semi) && !p.at(lex.RBrace) && !p.at(lex.EOF) {
		value = p.parseExpr()
	}
	return &ast.ReturnExpr{
		Span:  p.spanFrom(start),
		Value: value,
	}
}

// --- break ---

func (p *Parser) parseBreakExpr() *ast.BreakExpr {
	start := p.advance().Span // break
	var value ast.Expr
	if !p.at(lex.Semi) && !p.at(lex.RBrace) && !p.at(lex.EOF) {
		value = p.parseExpr()
	}
	return &ast.BreakExpr{
		Span:  p.spanFrom(start),
		Value: value,
	}
}

// --- spawn ---

func (p *Parser) parseSpawnExpr() *ast.SpawnExpr {
	start := p.advance().Span // spawn
	expr := p.parseExpr()
	return &ast.SpawnExpr{
		Span: p.spanFrom(start),
		Expr: expr,
	}
}

// --- closure ---

func (p *Parser) parseClosureExpr() *ast.ClosureExpr {
	start := p.advance().Span // fn

	p.expect(lex.LParen)
	params := p.parseParamList()
	p.expect(lex.RParen)

	var retType ast.TypeExpr
	if p.at(lex.Arrow) {
		p.advance()
		retType = p.parseTypeExpr()
	}

	body := p.parseBlock()

	return &ast.ClosureExpr{
		Span:       p.spanFrom(start),
		Params:     params,
		ReturnType: retType,
		Body:       body,
	}
}

// --- patterns ---

func (p *Parser) parsePattern() ast.Pattern {
	switch p.peekKind() {
	case lex.Ident:
		tok := p.advance()
		if tok.Literal == "_" {
			return &ast.WildcardPat{Span: tok.Span}
		}
		// Constructor pattern: Name(args) or Name { fields }
		if p.at(lex.LParen) {
			return p.parseConstructorPat(tok)
		}
		if p.at(lex.LBrace) {
			return p.parseStructPat(tok)
		}
		// Bare name: binding or unit variant (resolver decides)
		return &ast.BindPat{Span: tok.Span, Name: tok.Literal}

	case lex.KwTrue, lex.KwFalse, lex.KwNone, lex.KwSome:
		tok := p.advance()
		if tok.Kind == lex.KwSome && p.at(lex.LParen) {
			return p.parseConstructorPat(tok)
		}
		return &ast.LitPat{Span: tok.Span, Value: tok.Literal}

	case lex.IntLit, lex.FloatLit, lex.StringLit, lex.RawStringLit:
		tok := p.advance()
		return &ast.LitPat{Span: tok.Span, Value: tok.Literal}

	case lex.LParen:
		return p.parseTuplePat()

	default:
		p.errorf(p.peek().Span, "expected pattern, got %s", p.peekKind())
		tok := p.advance()
		return &ast.WildcardPat{Span: tok.Span}
	}
}

func (p *Parser) parseConstructorPat(name lex.Token) *ast.ConstructorPat {
	p.advance() // (
	var args []ast.Pattern
	for !p.at(lex.RParen) && !p.at(lex.EOF) {
		args = append(args, p.parsePattern())
		if !p.at(lex.RParen) {
			p.expect(lex.Comma)
		}
	}
	end := p.expect(lex.RParen)
	return &ast.ConstructorPat{
		Span: spanStartEnd(name.Span, end.Span),
		Name: name.Literal,
		Args: args,
	}
}

func (p *Parser) parseStructPat(name lex.Token) *ast.StructPat {
	p.advance() // {
	var fields []ast.FieldPat
	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		fstart := p.peek().Span
		fname := p.expect(lex.Ident)
		var pat ast.Pattern
		if p.at(lex.Colon) {
			p.advance()
			pat = p.parsePattern()
		}
		fields = append(fields, ast.FieldPat{
			Span: p.spanFrom(fstart),
			Name: fname.Literal,
			Pat:  pat,
		})
		if !p.at(lex.RBrace) {
			p.expect(lex.Comma)
		}
	}
	end := p.expect(lex.RBrace)
	return &ast.StructPat{
		Span:   spanStartEnd(name.Span, end.Span),
		Name:   name.Literal,
		Fields: fields,
	}
}

func (p *Parser) parseTuplePat() *ast.TuplePat {
	start := p.advance().Span // (
	var elems []ast.Pattern
	for !p.at(lex.RParen) && !p.at(lex.EOF) {
		elems = append(elems, p.parsePattern())
		if !p.at(lex.RParen) {
			p.expect(lex.Comma)
		}
	}
	end := p.expect(lex.RParen)
	return &ast.TuplePat{
		Span:  spanStartEnd(start, end.Span),
		Elems: elems,
	}
}

// --- span utilities ---

func spanBetween(a, b ast.Node) diagnostics.Span {
	sa := a.NodeSpan()
	sb := b.NodeSpan()
	return diagnostics.Span{File: sa.File, Start: sa.Start, End: sb.End}
}

func spanStartEnd(a diagnostics.Span, b diagnostics.Span) diagnostics.Span {
	return diagnostics.Span{File: a.File, Start: a.Start, End: b.End}
}

func spanToNode(a diagnostics.Span, b ast.Node) diagnostics.Span {
	sb := b.NodeSpan()
	return diagnostics.Span{File: a.File, Start: a.Start, End: sb.End}
}

func spanNodeTo(a ast.Node, b diagnostics.Span) diagnostics.Span {
	sa := a.NodeSpan()
	return diagnostics.Span{File: sa.File, Start: sa.Start, End: b.End}
}
