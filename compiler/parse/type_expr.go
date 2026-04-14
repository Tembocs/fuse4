package parse

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/lex"
)

// parseTypeExpr parses a type expression.
func (p *Parser) parseTypeExpr() ast.TypeExpr {
	switch p.peekKind() {
	case lex.LParen:
		return p.parseTupleOrUnitType()
	case lex.LBrack:
		return p.parseArrayOrSliceType()
	case lex.Ident, lex.KwSelfType:
		return p.parsePathType()
	default:
		p.errorf(p.peek().Span, "expected type expression, got %s", p.peekKind())
		tok := p.advance()
		return &ast.PathType{Span: tok.Span, Segments: []string{"<error>"}}
	}
}

// parseTupleOrUnitType: () or (T,) or (T, U) or (T)
func (p *Parser) parseTupleOrUnitType() ast.TypeExpr {
	start := p.advance().Span // (

	// unit type: ()
	if p.at(lex.RParen) {
		end := p.advance()
		return &ast.TupleType{
			Span:  spanStartEnd(start, end.Span),
			Elems: nil,
		}
	}

	first := p.parseTypeExpr()

	// tuple: (T, U, ...)
	if p.at(lex.Comma) {
		elems := []ast.TypeExpr{first}
		for p.at(lex.Comma) {
			p.advance()
			if p.at(lex.RParen) {
				break // trailing comma
			}
			elems = append(elems, p.parseTypeExpr())
		}
		end := p.expect(lex.RParen)
		return &ast.TupleType{
			Span:  spanStartEnd(start, end.Span),
			Elems: elems,
		}
	}

	// single-element parenthesized type (just returns the inner type)
	p.expect(lex.RParen)
	return first
}

// parseArrayOrSliceType: [T; N] or [T]
func (p *Parser) parseArrayOrSliceType() ast.TypeExpr {
	start := p.advance().Span // [

	// Ptr[T] is handled through path type + generic args.
	// This handles the literal [ in type position: [T] slice or [T; N] array.

	elem := p.parseTypeExpr()

	if p.at(lex.Semi) {
		// array: [T; N]
		p.advance() // ;
		size := p.parseExpr()
		end := p.expect(lex.RBrack)
		return &ast.ArrayType{
			Span: spanStartEnd(start, end.Span),
			Elem: elem,
			Size: size,
		}
	}

	// slice: [T]
	end := p.expect(lex.RBrack)
	return &ast.SliceType{
		Span: spanStartEnd(start, end.Span),
		Elem: elem,
	}
}

// parsePathType: Ident, Ident.Ident, Ident[T, U], Ptr[T]
func (p *Parser) parsePathType() ast.TypeExpr {
	start := p.peek().Span
	var segments []string

	first := p.advance() // Ident or Self
	segments = append(segments, first.Literal)

	// dotted path segments: core.list.List
	for p.at(lex.Dot) {
		p.advance() // .
		seg := p.expect(lex.Ident)
		segments = append(segments, seg.Literal)
	}

	// generic type args: [T, U]
	var typeArgs []ast.TypeExpr
	if p.at(lex.LBrack) {
		typeArgs = p.parseTypeArgList()
	}

	span := p.spanFrom(start)

	// Special case: Ptr[T] is a pointer type.
	if len(segments) == 1 && segments[0] == "Ptr" && len(typeArgs) == 1 {
		return &ast.PtrType{
			Span: span,
			Elem: typeArgs[0],
		}
	}

	return &ast.PathType{
		Span:     span,
		Segments: segments,
		TypeArgs: typeArgs,
	}
}

// parseTypeArgList parses [ T, U, ... ]
func (p *Parser) parseTypeArgList() []ast.TypeExpr {
	p.advance() // [
	var args []ast.TypeExpr
	for !p.at(lex.RBrack) && !p.at(lex.EOF) {
		args = append(args, p.parseTypeExpr())
		if !p.at(lex.RBrack) {
			p.expect(lex.Comma)
		}
	}
	p.expect(lex.RBrack)
	return args
}

