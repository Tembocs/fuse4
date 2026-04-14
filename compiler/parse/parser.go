// Package parse owns parsing from tokens to AST.
package parse

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/lex"
)

// Parser is a recursive-descent parser for Fuse source files.
type Parser struct {
	tokens []lex.Token
	pos    int
	errors []diagnostics.Diagnostic
}

// Parse tokenizes src and parses it into an AST.
func Parse(filename string, src []byte) (*ast.File, []diagnostics.Diagnostic) {
	l := lex.New(filename, src)
	tokens, lexErrs := l.Tokenize()

	p := &Parser{tokens: tokens}
	file := p.parseFile()

	errs := make([]diagnostics.Diagnostic, 0, len(lexErrs)+len(p.errors))
	errs = append(errs, lexErrs...)
	errs = append(errs, p.errors...)
	return file, errs
}

// --- token access ---

func (p *Parser) peek() lex.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lex.Token{Kind: lex.EOF}
}

func (p *Parser) peekKind() lex.TokenKind {
	return p.peek().Kind
}

// peekAt returns the token at pos+offset without advancing.
func (p *Parser) peekAt(offset int) lex.Token {
	idx := p.pos + offset
	if idx >= 0 && idx < len(p.tokens) {
		return p.tokens[idx]
	}
	return lex.Token{Kind: lex.EOF}
}

func (p *Parser) advance() lex.Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) expect(kind lex.TokenKind) lex.Token {
	tok := p.peek()
	if tok.Kind != kind {
		p.errorf(tok.Span, "expected %s, got %s", kind, tok.Kind)
		// Do NOT advance — let the caller or recovery handle progress.
		// Return a synthetic token so callers get a usable value.
		return tok
	}
	return p.advance()
}

// expectIdentLike expects an identifier but also accepts keywords that are
// valid as names in certain contexts (import path segments, variant names).
func (p *Parser) expectIdentLike() lex.Token {
	tok := p.peek()
	if tok.Kind == lex.Ident || tok.IsKeyword() {
		return p.advance()
	}
	p.errorf(tok.Span, "expected identifier, got %s", tok.Kind)
	return tok
}

func (p *Parser) match(kind lex.TokenKind) (lex.Token, bool) {
	if p.peekKind() == kind {
		return p.advance(), true
	}
	return lex.Token{}, false
}

func (p *Parser) at(kind lex.TokenKind) bool {
	return p.peekKind() == kind
}

func (p *Parser) atAny(kinds ...lex.TokenKind) bool {
	k := p.peekKind()
	for _, want := range kinds {
		if k == want {
			return true
		}
	}
	return false
}

// --- diagnostics ---

func (p *Parser) errorf(span diagnostics.Span, format string, args ...any) {
	p.errors = append(p.errors, diagnostics.Errorf(span, format, args...))
}

// --- recovery ---

// synchronize skips tokens until it reaches one of the given token kinds
// or a likely item-start keyword. Used for error recovery.
func (p *Parser) synchronize() {
	for !p.at(lex.EOF) {
		switch p.peekKind() {
		case lex.KwFn, lex.KwPub, lex.KwStruct, lex.KwEnum,
			lex.KwTrait, lex.KwImpl, lex.KwConst, lex.KwType,
			lex.KwExtern, lex.KwImport:
			return
		case lex.Semi, lex.RBrace:
			p.advance()
			return
		}
		p.advance()
	}
}

// --- span helpers ---

func (p *Parser) spanFrom(start diagnostics.Span) diagnostics.Span {
	end := start
	if p.pos > 0 && p.pos-1 < len(p.tokens) {
		end = p.tokens[p.pos-1].Span
	}
	return diagnostics.Span{
		File:  start.File,
		Start: start.Start,
		End:   end.End,
	}
}

// --- generic params ---

// parseGenericParams parses [ T, U : Bound, ... ] if present.
func (p *Parser) parseGenericParams() []ast.GenericParam {
	if !p.at(lex.LBrack) {
		return nil
	}
	p.advance() // [

	var params []ast.GenericParam
	for !p.at(lex.RBrack) && !p.at(lex.EOF) {
		start := p.peek().Span
		name := p.expect(lex.Ident)

		var bounds []ast.TypeExpr
		if p.at(lex.Colon) {
			p.advance() // :
			bounds = append(bounds, p.parseTypeExpr())
			for p.at(lex.Plus) {
				p.advance()
				bounds = append(bounds, p.parseTypeExpr())
			}
		}

		params = append(params, ast.GenericParam{
			Span:   p.spanFrom(start),
			Name:   name.Literal,
			Bounds: bounds,
		})

		if !p.at(lex.RBrack) {
			p.expect(lex.Comma)
		}
	}
	p.expect(lex.RBrack)
	return params
}

// --- where clause ---

func (p *Parser) parseWhereClause() *ast.WhereClause {
	if !p.at(lex.KwWhere) {
		return nil
	}
	start := p.advance().Span // where

	var constraints []ast.WhereConstraint
	for {
		cs := p.peek().Span
		ty := p.parseTypeExpr()
		p.expect(lex.Colon)
		var bounds []ast.TypeExpr
		bounds = append(bounds, p.parseTypeExpr())
		for p.at(lex.Plus) {
			p.advance()
			bounds = append(bounds, p.parseTypeExpr())
		}
		constraints = append(constraints, ast.WhereConstraint{
			Span:   p.spanFrom(cs),
			Type:   ty,
			Bounds: bounds,
		})
		if !p.at(lex.Comma) {
			break
		}
		p.advance() // ,
	}

	return &ast.WhereClause{
		Span:        p.spanFrom(start),
		Constraints: constraints,
	}
}

// --- param list ---

func (p *Parser) parseParamList() []ast.Param {
	var params []ast.Param
	for !p.at(lex.RParen) && !p.at(lex.EOF) {
		start := p.peek().Span
		ownership := lex.TokenKind(0)

		// receiver forms: self, ref self, mutref self, owned self
		if p.atAny(lex.KwRef, lex.KwMutref, lex.KwOwned) && p.peekAt(1).Kind == lex.KwSelfValue {
			ownership = p.advance().Kind
			name := p.advance() // self
			params = append(params, ast.Param{
				Span:      p.spanFrom(start),
				Ownership: ownership,
				Name:      name.Literal,
			})
			if !p.at(lex.RParen) {
				p.expect(lex.Comma)
			}
			continue
		}
		if p.at(lex.KwSelfValue) {
			name := p.advance()
			params = append(params, ast.Param{
				Span: p.spanFrom(start),
				Name: name.Literal,
			})
			if !p.at(lex.RParen) {
				p.expect(lex.Comma)
			}
			continue
		}

		// normal param: IDENT ":" [ ownership ] type_expr
		nameTok := p.expect(lex.Ident)
		p.expect(lex.Colon)

		// ownership modifier between colon and type
		if p.atAny(lex.KwRef, lex.KwMutref, lex.KwOwned) {
			ownership = p.advance().Kind
		}

		ty := p.parseTypeExpr()

		params = append(params, ast.Param{
			Span:      p.spanFrom(start),
			Ownership: ownership,
			Name:      nameTok.Literal,
			Type:      ty,
		})

		if !p.at(lex.RParen) {
			p.expect(lex.Comma)
		}
	}
	return params
}

// --- field list ---

func (p *Parser) parseFieldList() []ast.Field {
	var fields []ast.Field
	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		if p.peekKind() != lex.Ident {
			break // not a field — bail
		}
		start := p.peek().Span
		name := p.advance() // ident
		p.expect(lex.Colon)
		ty := p.parseTypeExpr()

		fields = append(fields, ast.Field{
			Span: p.spanFrom(start),
			Name: name.Literal,
			Type: ty,
		})

		if !p.at(lex.RBrace) {
			if _, ok := p.match(lex.Comma); !ok {
				break
			}
		}
	}
	return fields
}
