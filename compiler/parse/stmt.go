package parse

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/lex"
)

// parseBlock parses a block expression: { stmts... [tail_expr] }
func (p *Parser) parseBlock() *ast.BlockExpr {
	start := p.expect(lex.LBrace).Span
	var stmts []ast.Stmt
	var tail ast.Expr

	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		// item declarations inside blocks
		if p.isItemStart() {
			item := p.parseItem()
			if item != nil {
				stmts = append(stmts, &ast.ItemStmt{
					Span: item.NodeSpan(),
					Item: item,
				})
			}
			continue
		}

		// let / var statements
		if p.at(lex.KwLet) {
			stmts = append(stmts, p.parseLetStmt())
			continue
		}
		if p.at(lex.KwVar) {
			stmts = append(stmts, p.parseVarStmt())
			continue
		}

		// expression or expression-statement
		expr := p.parseExpr()

		// If followed by ';', it's an expression statement.
		// If followed by '}', it's the tail expression.
		if p.at(lex.Semi) {
			p.advance() // ;
			stmts = append(stmts, &ast.ExprStmt{
				Span: expr.NodeSpan(),
				Expr: expr,
			})
		} else if p.at(lex.RBrace) {
			tail = expr
		} else {
			// No semicolon and not at '}': implicit statement end
			// (this handles things like if/match/block exprs without ';')
			stmts = append(stmts, &ast.ExprStmt{
				Span: expr.NodeSpan(),
				Expr: expr,
			})
		}
	}

	end := p.expect(lex.RBrace)
	return &ast.BlockExpr{
		Span:  spanStartEnd(start, end.Span),
		Stmts: stmts,
		Tail:  tail,
	}
}

func (p *Parser) isItemStart() bool {
	k := p.peekKind()
	switch k {
	case lex.KwFn:
		// fn IDENT is a declaration; fn ( is a closure expression
		return p.peekAt(1).Kind == lex.Ident
	case lex.KwStruct, lex.KwEnum, lex.KwTrait,
		lex.KwImpl, lex.KwConst, lex.KwType, lex.KwExtern:
		return true
	case lex.KwPub:
		next := p.peekAt(1).Kind
		return next == lex.KwFn || next == lex.KwStruct || next == lex.KwEnum ||
			next == lex.KwTrait || next == lex.KwConst || next == lex.KwType ||
			next == lex.KwExtern
	case lex.At:
		return true // decorator starts an item
	}
	return false
}

func (p *Parser) parseLetStmt() *ast.LetStmt {
	start := p.advance().Span // let
	name := p.expect(lex.Ident)

	var ty ast.TypeExpr
	if p.at(lex.Colon) {
		p.advance()
		ty = p.parseTypeExpr()
	}

	var value ast.Expr
	if p.at(lex.Eq) {
		p.advance()
		value = p.parseExpr()
	}

	p.expect(lex.Semi)
	return &ast.LetStmt{
		Span:  p.spanFrom(start),
		Name:  name.Literal,
		Type:  ty,
		Value: value,
	}
}

func (p *Parser) parseVarStmt() *ast.VarStmt {
	start := p.advance().Span // var
	name := p.expect(lex.Ident)

	var ty ast.TypeExpr
	if p.at(lex.Colon) {
		p.advance()
		ty = p.parseTypeExpr()
	}

	var value ast.Expr
	if p.at(lex.Eq) {
		p.advance()
		value = p.parseExpr()
	}

	p.expect(lex.Semi)
	return &ast.VarStmt{
		Span:  p.spanFrom(start),
		Name:  name.Literal,
		Type:  ty,
		Value: value,
	}
}
