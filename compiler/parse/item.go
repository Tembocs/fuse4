package parse

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/lex"
)

// parseFile parses a complete source file.
func (p *Parser) parseFile() *ast.File {
	start := p.peek().Span
	var items []ast.Item

	for !p.at(lex.EOF) {
		item := p.parseTopLevelItem()
		if item != nil {
			items = append(items, item)
		}
	}

	return &ast.File{
		Span:  p.spanFrom(start),
		Items: items,
	}
}

func (p *Parser) parseTopLevelItem() ast.Item {
	switch p.peekKind() {
	case lex.KwImport:
		return p.parseImport()
	default:
		return p.parseItem()
	}
}

// parseItem parses an item declaration. It handles optional pub and decorators.
func (p *Parser) parseItem() ast.Item {
	// collect decorators
	var decorators []ast.Decorator
	for p.at(lex.At) {
		decorators = append(decorators, p.parseDecorator())
	}

	pub := false
	if _, ok := p.match(lex.KwPub); ok {
		pub = true
	}

	switch p.peekKind() {
	case lex.KwFn:
		return p.parseFnDecl(pub)
	case lex.KwStruct:
		return p.parseStructDecl(pub, decorators)
	case lex.KwEnum:
		return p.parseEnumDecl(pub)
	case lex.KwTrait:
		return p.parseTraitDecl(pub)
	case lex.KwImpl:
		return p.parseImplDecl()
	case lex.KwConst:
		return p.parseConstDecl(pub)
	case lex.KwType:
		return p.parseTypeAliasDecl(pub)
	case lex.KwExtern:
		return p.parseExternDecl(pub)
	default:
		p.errorf(p.peek().Span, "expected item declaration, got %s", p.peekKind())
		p.synchronize()
		return nil
	}
}

func (p *Parser) parseDecorator() ast.Decorator {
	start := p.advance().Span // @
	name := p.expect(lex.Ident)
	var args []ast.Expr
	if p.at(lex.LParen) {
		p.advance() // (
		for !p.at(lex.RParen) && !p.at(lex.EOF) {
			args = append(args, p.parseExpr())
			if !p.at(lex.RParen) {
				p.expect(lex.Comma)
			}
		}
		p.expect(lex.RParen)
	}
	return ast.Decorator{
		Span: p.spanFrom(start),
		Name: name.Literal,
		Args: args,
	}
}

// --- import ---

func (p *Parser) parseImport() *ast.ImportDecl {
	start := p.advance().Span // import

	var path []string
	nameTok := p.expectIdentLike()
	path = append(path, nameTok.Literal)
	for p.at(lex.Dot) {
		p.advance() // .
		seg := p.expectIdentLike()
		path = append(path, seg.Literal)
	}

	alias := ""
	if p.at(lex.KwAs) {
		p.advance() // as
		aliasTok := p.expect(lex.Ident)
		alias = aliasTok.Literal
	}

	p.expect(lex.Semi)

	return &ast.ImportDecl{
		Span:  p.spanFrom(start),
		Path:  path,
		Alias: alias,
	}
}

// --- function ---

func (p *Parser) parseFnDecl(pub bool) *ast.FnDecl {
	start := p.peek().Span
	p.advance() // fn

	name := p.expect(lex.Ident)
	gparams := p.parseGenericParams()

	p.expect(lex.LParen)
	params := p.parseParamList()
	p.expect(lex.RParen)

	var retType ast.TypeExpr
	if p.at(lex.Arrow) {
		p.advance() // ->
		retType = p.parseTypeExpr()
	}

	where := p.parseWhereClause()
	body := p.parseBlock()

	return &ast.FnDecl{
		Span:          p.spanFrom(start),
		Public:        pub,
		Name:          name.Literal,
		GenericParams: gparams,
		Params:        params,
		ReturnType:    retType,
		Where:         where,
		Body:          body,
	}
}

// --- struct ---

func (p *Parser) parseStructDecl(pub bool, decorators []ast.Decorator) *ast.StructDecl {
	start := p.peek().Span
	p.advance() // struct

	name := p.expect(lex.Ident)
	gparams := p.parseGenericParams()

	p.expect(lex.LBrace)
	fields := p.parseFieldList()
	p.expect(lex.RBrace)

	return &ast.StructDecl{
		Span:          p.spanFrom(start),
		Public:        pub,
		Decorators:    decorators,
		Name:          name.Literal,
		GenericParams: gparams,
		Fields:        fields,
	}
}

// --- enum ---

func (p *Parser) parseEnumDecl(pub bool) *ast.EnumDecl {
	start := p.peek().Span
	p.advance() // enum

	name := p.expect(lex.Ident)
	gparams := p.parseGenericParams()

	p.expect(lex.LBrace)
	var variants []ast.Variant
	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		variants = append(variants, p.parseVariant())
		if !p.at(lex.RBrace) {
			p.expect(lex.Comma)
		}
	}
	p.expect(lex.RBrace)

	return &ast.EnumDecl{
		Span:          p.spanFrom(start),
		Public:        pub,
		Name:          name.Literal,
		GenericParams: gparams,
		Variants:      variants,
	}
}

func (p *Parser) parseVariant() ast.Variant {
	start := p.peek().Span
	name := p.expectIdentLike()

	switch p.peekKind() {
	case lex.LParen:
		// tuple variant: Name(T1, T2)
		p.advance() // (
		var types []ast.TypeExpr
		for !p.at(lex.RParen) && !p.at(lex.EOF) {
			types = append(types, p.parseTypeExpr())
			if !p.at(lex.RParen) {
				p.expect(lex.Comma)
			}
		}
		p.expect(lex.RParen)
		return ast.Variant{
			Span:  p.spanFrom(start),
			Name:  name.Literal,
			Kind:  ast.VariantTuple,
			Types: types,
		}

	case lex.LBrace:
		// struct variant: Name { a: T1, b: T2 }
		p.advance() // {
		fields := p.parseFieldList()
		p.expect(lex.RBrace)
		return ast.Variant{
			Span:   p.spanFrom(start),
			Name:   name.Literal,
			Kind:   ast.VariantStruct,
			Fields: fields,
		}

	default:
		// unit variant: Name
		return ast.Variant{
			Span: p.spanFrom(start),
			Name: name.Literal,
			Kind: ast.VariantUnit,
		}
	}
}

// --- trait ---

func (p *Parser) parseTraitDecl(pub bool) *ast.TraitDecl {
	start := p.peek().Span
	p.advance() // trait

	name := p.expect(lex.Ident)
	gparams := p.parseGenericParams()

	// supertraits: trait Foo : Bar + Baz { ... }
	var supers []ast.TypeExpr
	if p.at(lex.Colon) {
		p.advance() // :
		supers = append(supers, p.parseTypeExpr())
		for p.at(lex.Plus) {
			p.advance()
			supers = append(supers, p.parseTypeExpr())
		}
	}

	p.expect(lex.LBrace)
	var items []ast.Item
	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		item := p.parseTraitItem()
		if item != nil {
			items = append(items, item)
		}
	}
	p.expect(lex.RBrace)

	return &ast.TraitDecl{
		Span:          p.spanFrom(start),
		Public:        pub,
		Name:          name.Literal,
		GenericParams: gparams,
		Supertraits:   supers,
		Items:         items,
	}
}

func (p *Parser) parseTraitItem() ast.Item {
	// trait items are method signatures (fn with optional body)
	if p.at(lex.KwFn) {
		return p.parseFnDeclOrSig(false)
	}
	p.errorf(p.peek().Span, "expected trait item, got %s", p.peekKind())
	p.synchronize()
	return nil
}

// parseFnDeclOrSig parses either a full fn with body or just a signature
// (for trait items that may omit the body).
func (p *Parser) parseFnDeclOrSig(pub bool) *ast.FnDecl {
	start := p.peek().Span
	p.advance() // fn

	name := p.expect(lex.Ident)
	gparams := p.parseGenericParams()

	p.expect(lex.LParen)
	params := p.parseParamList()
	p.expect(lex.RParen)

	var retType ast.TypeExpr
	if p.at(lex.Arrow) {
		p.advance()
		retType = p.parseTypeExpr()
	}

	where := p.parseWhereClause()

	var body *ast.BlockExpr
	if p.at(lex.LBrace) {
		body = p.parseBlock()
	} else {
		p.expect(lex.Semi) // signature-only
	}

	return &ast.FnDecl{
		Span:          p.spanFrom(start),
		Public:        pub,
		Name:          name.Literal,
		GenericParams: gparams,
		Params:        params,
		ReturnType:    retType,
		Where:         where,
		Body:          body,
	}
}

// --- impl ---

func (p *Parser) parseImplDecl() *ast.ImplDecl {
	start := p.advance().Span // impl

	gparams := p.parseGenericParams()
	target := p.parseTypeExpr()

	var traitType ast.TypeExpr
	if p.at(lex.Colon) {
		p.advance() // :
		// The first type was actually the trait; swap.
		traitType = target
		target = p.parseTypeExpr()
	}

	where := p.parseWhereClause()

	p.expect(lex.LBrace)
	var items []ast.Item
	for !p.at(lex.RBrace) && !p.at(lex.EOF) {
		pub := false
		if _, ok := p.match(lex.KwPub); ok {
			pub = true
		}
		if p.at(lex.KwFn) {
			items = append(items, p.parseFnDecl(pub))
		} else {
			p.errorf(p.peek().Span, "expected method in impl block, got %s", p.peekKind())
			p.synchronize()
		}
	}
	p.expect(lex.RBrace)

	return &ast.ImplDecl{
		Span:          p.spanFrom(start),
		GenericParams: gparams,
		Target:        target,
		Trait:         traitType,
		Where:         where,
		Items:         items,
	}
}

// --- const ---

func (p *Parser) parseConstDecl(pub bool) *ast.ConstDecl {
	start := p.peek().Span
	p.advance() // const

	name := p.expect(lex.Ident)

	var ty ast.TypeExpr
	if p.at(lex.Colon) {
		p.advance()
		ty = p.parseTypeExpr()
	}

	p.expect(lex.Eq)
	value := p.parseExpr()
	p.expect(lex.Semi)

	return &ast.ConstDecl{
		Span:   p.spanFrom(start),
		Public: pub,
		Name:   name.Literal,
		Type:   ty,
		Value:  value,
	}
}

// --- type alias ---

func (p *Parser) parseTypeAliasDecl(pub bool) *ast.TypeAliasDecl {
	start := p.peek().Span
	p.advance() // type

	name := p.expect(lex.Ident)
	gparams := p.parseGenericParams()
	p.expect(lex.Eq)
	ty := p.parseTypeExpr()
	p.expect(lex.Semi)

	return &ast.TypeAliasDecl{
		Span:          p.spanFrom(start),
		Public:        pub,
		Name:          name.Literal,
		GenericParams: gparams,
		Type:          ty,
	}
}

// --- extern ---

func (p *Parser) parseExternDecl(pub bool) *ast.ExternFnDecl {
	start := p.peek().Span
	p.advance() // extern

	p.expect(lex.KwFn)
	name := p.expect(lex.Ident)

	p.expect(lex.LParen)
	params := p.parseParamList()
	p.expect(lex.RParen)

	var retType ast.TypeExpr
	if p.at(lex.Arrow) {
		p.advance()
		retType = p.parseTypeExpr()
	}

	p.expect(lex.Semi)

	return &ast.ExternFnDecl{
		Span:       p.spanFrom(start),
		Public:     pub,
		Name:       name.Literal,
		Params:     params,
		ReturnType: retType,
	}
}
