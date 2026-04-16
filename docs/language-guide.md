# Fuse Language Guide

> Status: normative for the next production attempt of Fuse (`fuse4`).
>
> This document is the specification of the Fuse language. It is written for
> two audiences at once:
>
> - users, who need to understand how to write correct Fuse programs
> - implementers, who must be able to build a correct compiler from this
>   document alone
>
> Every section that defines a language feature includes an implementation
> contract. If the implementation contract is ambiguous, the document is
> defective.

## Table of contents

1. Introduction
2. Lexical structure
3. Types
4. Ownership and lifetimes
5. Expressions
6. Statements and control flow
7. Functions and methods
8. Structs and enums
9. Traits
10. Generics and monomorphization
11. Modules and imports
12. Concurrency
13. Error handling
14. The unsafe boundary
15. The C interop ABI
16. Backend representation contracts
17. Grammar reference (EBNF)

## 1. Introduction

Fuse is a compiled, statically typed systems programming language with four
defining properties.

1. Memory safety without a garbage collector.
2. Concurrency safety without a borrow checker.
3. Developer experience as a first-class constraint.
4. Self-hosting as the terminal compiler goal.

Fuse is built in three stages.

- Stage 0 (`fusec0`) was a Python interpreter and is retired.
- Stage 1 (`fusec`) is a Go compiler that emits C11 as a bootstrap backend.
- Stage 2 is the Fuse compiler written in Fuse and compiled by Stage 1.

The implementation stack during bootstrap is fixed.

- Go remains the host implementation language for Stage 1.
- C remains the runtime language during bootstrap.
- Fuse is the implementation language of Stage 2.

The C11 backend is a bootstrap strategy, not the terminal architecture. After
Fuse compiles itself reliably and the native backend is stable, Go and C are
retired from the compiler implementation path. The language semantics are not
defined in terms of Go or C; those are temporary implementation tools.

Fuse v1 is frozen. Features not described in this document do not exist.

### Example

```fuse
import core.result.Result;
import full.chan.Chan;

@value struct WorkItem {
		id: U64,
		payload: String,
}

pub fn process(queue: mutref Chan[WorkItem]) -> Result[(), String] {
		let item = queue.recv()?;
		let upper = item.payload.toUpper();
		spawn fn() {
				log(upper);
		};
		return Ok(());
}
```

The example demonstrates three core properties.

- mutation is visible at the call site through `mutref`
- error propagation is visible through `?`
- concurrency is explicit through `spawn`

## 2. Lexical structure

Fuse source files are UTF-8 text. A BOM is not permitted. Line endings may be
LF or CRLF and are normalized by the lexer.

### 2.1 Tokens

The lexer emits the following token classes.

- identifiers
- keywords
- integer literals
- float literals
- string literals
- raw string literals
- punctuation and delimiters
- operators
- comments and whitespace, which are discarded except where needed for spans

### 2.2 Identifiers

An identifier starts with a Unicode letter or `_`, and continues with Unicode
letters, Unicode digits, or `_`. Identifiers are case-sensitive.

### 2.3 Keywords

The full active keyword set is:

`fn`, `pub`, `struct`, `enum`, `trait`, `impl`, `for`, `in`, `while`, `loop`,
`if`, `else`, `match`, `return`, `let`, `var`, `move`, `ref`, `mutref`,
`owned`, `unsafe`, `spawn`, `chan`, `import`, `as`, `mod`, `use`, `type`,
`const`, `static`, `extern`, `break`, `continue`, `where`, `Self`, `self`,
`true`, `false`, `None`, `Some`.

Additional words may be reserved for future use, but they are not active unless
listed above.

### 2.4 Integer literals

Integer literals may be:

- decimal: `0`, `42`, `9001`
- hexadecimal: `0x2A`
- octal: `0o52`
- binary: `0b101010`

An integer literal may carry a suffix:

`i8`, `i16`, `i32`, `i64`, `i128`, `isize`, `u8`, `u16`, `u32`, `u64`,
`u128`, `usize`.

Examples:

```fuse
64usize
0xffu8
0b1010i32
```

### 2.5 Float literals

Float literals may contain a decimal point, an exponent, or both. Valid suffixes
are `f32` and `f64`.

Examples:

```fuse
1.0
6.02e23
3.14f32
```

### 2.6 String literals

String literals are UTF-8 and support standard escape sequences.

Supported escapes include:

- `\n`
- `\r`
- `\t`
- `\\`
- `\"`
- Unicode escapes as defined by the compiler implementation

### 2.7 Raw string literals

Raw strings use the Rust-style form:

- `r"..."`
- `r#"..."#`
- `r##"..."##`

The number of `#` characters must match between opener and closer.

### 2.8 Operators and punctuation

Fuse includes the usual arithmetic, logical, assignment, indexing, field access,
and delimiter tokens. The token `?.` is a single longest-match token used for
optional chaining.

### 2.9 Comments

- Line comments begin with `//` and continue to end of line.
- Block comments use `/* ... */` and may nest.

### Implementation contracts

#### Raw string recognition

A raw string is recognized only when the lexer matches the full prefix pattern
`r#*"` and the corresponding closing `"#*` sequence exists. The lexer must not
enter raw-string mode on `r` followed by `#` alone.

`r#abc` must tokenize as:

```text
IDENT("r")
#
IDENT("abc")
```

#### `?.` longest-match rule

The lexer emits `?.` as one token. It is not `?` followed by `.`. The parser
must interpret `expr?.field` as optional chaining, not as postfix `?` applied
to `expr` followed by field access on the result.

#### Literal normalization

Literal text must be normalized at the HIR-to-MIR boundary.

- Integer suffixes are stripped before MIR constant emission.
- String literal payloads are stored without their surrounding quote characters.

The generated C must never contain raw Fuse literal spellings such as
`INT32_C(64usize)` or doubled string quotes such as `""NaN""`.

## 3. Types

### 3.1 Primitive types

Fuse defines the following primitive numeric and scalar types.

- `Bool`
- `Char`
- `I8`, `I16`, `I32`, `I64`, `I128`, `ISize`
- `U8`, `U16`, `U32`, `U64`, `U128`, `USize`
- `F32`, `F64`

Two aliases are part of the language surface.

- `Int` is the platform word-sized signed integer and is equivalent to `ISize`.
- `Float` is the default floating-point type and is equivalent to `F64`.

### 3.2 Compound types

Fuse supports the following compound types.

- unit: `()`
- tuples: `(T1, T2, ...)`
- arrays: `[T; N]`
- slices: `[T]`
- user-defined structs and enums

### 3.3 Pointer type

`Ptr[T]` is a raw pointer type.

- it is not a borrow
- it does not participate in ownership analysis
- it is permitted only in `unsafe` contexts or runtime bridge code

### 3.4 Generic types

Generic types include standard-library types such as:

- `List[T]`
- `Option[T]`
- `Result[T, E]`

and user-defined generic types.

### 3.5 Type identity

Nominal types are compared by both name and defining symbol. Two types with the
same name from different modules are different types.

Examples:

- `ast.Expr` is not `hir.Expr`
- `resolve.SymbolId` is not `hir.SymbolId`

### 3.6 Primitive method surface

Primitive types expose methods even when those methods are not declared in
ordinary user source.

Required built-in method surface:

- integer types: `toFloat() -> F64`, `toInt() -> I64`, `abs() -> Self`,
	`min(Self) -> Self`, `max(Self) -> Self`
- `F32`, `F64`: `toInt() -> I64`, `isNan() -> Bool`,
	`isInfinite() -> Bool`, `floor() -> Self`, `ceil() -> Self`,
	`sqrt() -> Self`, `abs() -> Self`
- `Char`: `toInt() -> U32`, `isLetter() -> Bool`, `isDigit() -> Bool`,
	`isWhitespace() -> Bool`
- `Bool`: `not() -> Bool`

### 3.7 Numeric operators

Arithmetic operators work on numeric types. Comparison and equality operators
are defined over types that implement the required traits or intrinsic rules.

### 3.8 Tuple field access

Tuple elements are accessed by decimal index.

Examples:

```fuse
let p = (1, 2);
let x = p.0;
let y = p.1;
```

### Implementation contracts

#### Type identity rule

Two nominal types are the same type if and only if they share:

- the same declared name, and
- the same defining symbol or defining module identity

Name-only equality is invalid in a multi-module compiler.

#### Primitive method registration

The checker must register the primitive method surface before any function body
checking begins. Primitive method lookup must not depend on user-declared impls.

#### Numeric widening

Binary operators between two numeric types in the same family are permitted when
the wider type can represent all values of the narrower. For example,
`I32 == ISize` is legal. Bitwise operators require matching signedness and width.

#### Tuple numeric fields

When the receiver is a tuple type, a decimal field name such as `0` or `1` is
an index into the tuple, not a struct field name. This is the only legal tuple
field access form.

## 4. Ownership and lifetimes

Fuse uses explicit ownership annotations and a single liveness computation to
guarantee deterministic destruction without a tracing garbage collector.

### 4.1 Ownership forms

Fuse exposes four ownership forms.

- value ownership: ordinary by-value binding or parameter
- `ref`: shared borrow
- `mutref`: mutable borrow
- `owned`: transferring ownership into a callee or binding

### 4.2 Move semantics

Moves are explicit when needed to escape the normal ownership flow. After a move,
the moved value is no longer usable. Later passes must treat any use after move
as invalid.

### 4.3 Last use and destruction

Destruction is deterministic. The compiler computes liveness once and inserts
drop behavior according to ownership and last-use information.

### 4.4 Escape rules

A borrowed value may not outlive its owner. Escaping borrows are rejected unless
the language construct explicitly transfers ownership or performs an allowed move.

### Implementation contracts

#### Implicit `mutref` on mutable receivers

When a method receiver is declared as `mutref self`, the call site does not need
to spell `mutref` explicitly if the receiver is an existing mutable binding.

Example:

```fuse
var items = List[Int].new();
items.push(1);
```

The call to `push` is valid without `mutref items` because `items` is already a
mutable binding.

#### Borrow lowering rule

`ref x` and `mutref x` must lower to `InstrBorrow` with a precise borrow kind.
They must not lower to a generic unary operator representation. The backend must
be able to distinguish borrow formation from other unary expressions.

## 5. Expressions

### 5.1 Basic expressions

Fuse expressions include:

- literals
- identifiers and paths
- tuples
- blocks
- field access
- indexing
- calls
- `if` expressions
- `match` expressions
- loops where the construct yields a value
- closures, if enabled by the language version

### 5.2 Equality and ordering

`==` and `!=` are semantic operations, not merely textual C operators. Equality
uses trait-driven or compiler-defined semantics. Ordering operators require the
appropriate comparison semantics for the operand type.

### 5.3 Optional chaining

Optional chaining uses `?.` and propagates failure or absence according to the
operand type and language rule of the surrounding expression.

### 5.4 Struct literals

Struct literals use the syntax:

```fuse
Point { x: 1.0, y: 2.0 }
```

### 5.5 Enum variant expressions

Enum variants may appear in bare form when unambiguous or in qualified form as
`EnumName.Variant`.

### Implementation contracts

#### Field access versus method-call disambiguation

The surface syntax `obj.name` is ambiguous between a data field read and a
method reference. The lowerer must disambiguate using position.

- If `obj.name` is the direct callee of a call expression, treat it as a method
	reference and emit a method call with `obj` as the first argument.
- Otherwise, treat it as a field access and emit the appropriate field-address
	or field-read logic.

Lowering `self.len()` as a field access is a backend bug.

#### Struct literal disambiguation

`IDENT {` is not automatically a struct literal. It is a struct literal only if
the brace body syntactically looks like a field list, either empty or beginning
with `IDENT :`. Otherwise the identifier remains an expression and the `{` opens
the surrounding block.

#### Qualified enum variant access

The resolver must support `EnumName.Variant` in expression position. Enum
variants are hoisted to module scope, so qualified access is required to make
non-trivial code unambiguous.

#### Generic inference from context

When explicit type arguments are absent, generic arguments are inferred from:

1. value argument types
2. the expected result type from the surrounding expression context

`List.new()` in a field typed `List[Expr]` must infer `Expr` from context.

#### Explicit type arguments on zero-argument generic calls

Calls such as `sizeOf[T]()` and `marker[Int]()` carry type information even when
they have no value arguments. The monomorphizer must receive explicit callee
type arguments directly; a value-argument-only inference path is incomplete.

#### Equality lowering rule

The lowerer must not emit raw backend equality for every type. For non-scalar
types, equality must lower through the type's equality semantics rather than a
plain C `==` or `!=`.

## 6. Statements and control flow

### 6.1 Statements

Fuse statements include:

- `let` bindings
- `var` bindings
- assignments
- item declarations inside blocks where the grammar allows them
- expression statements

### 6.2 Control flow

Fuse supports:

- `if` / `else`
- `while`
- `loop`
- `for ... in`
- `match`
- `return`
- `break`
- `continue`

### 6.3 Diverging expressions

Expressions that never return have type `!` or `Never`. Calls to panic-like or
abort-like functions are diverging expressions.

### Implementation contracts

#### Divergence is structural

After a diverging call, there is no continuing control-flow path. MIR must model
that structurally. The code generator must not synthesize fake temporaries or
fallback values that assume execution continues.

#### Sealed control-flow blocks

Lowering of `return`, `break`, and `continue` must seal the current basic block.
Later control-flow construction must not treat the sealed block as a reachable
fallthrough predecessor.

## 7. Functions and methods

### 7.1 Function declarations

Functions are declared with `fn` and may be `pub`. Parameters and the return
type are explicit.

Example:

```fuse
pub fn parse(input: ref String) -> Result[Expr, ParseError] {
		...
}
```

### 7.2 Methods and receivers

Methods appear inside impl blocks. Receivers may be:

- `self`
- `ref self`
- `mutref self`
- `owned self`

Associated functions have no receiver.

### 7.3 Extern declarations

Extern declarations describe FFI entry points and do not have Fuse bodies.

### Implementation contracts

#### Function type registration pre-pass

Every function declaration node, including impl methods and extern declarations,
must receive its function type before any function body is checked.

The checker therefore requires two passes.

- Pass 1: register all function types
- Pass 2: check all function bodies

If impl methods only receive their type during body checking, their metadata
remains `Unknown` during lowering and code generation, which corrupts the
backend pipeline.

#### No function-type gaps after checking

After successful checking, no function declaration in checked HIR may retain an
unknown function type. This is a hard invariant, not a best-effort rule.

## 8. Structs and enums

### 8.1 Structs

Fuse supports plain structs and value-oriented structs. A value-oriented struct
may opt into auto-derived behavior for the core trait set.

### 8.2 Enums

Enums may contain:

- unit variants
- tuple-like variants
- struct-like variants

### 8.3 Pattern matching

Enum variants are destructured through `match` patterns.

### Implementation contracts

#### Enum variant hoisting

Enum variants are hoisted into the enclosing module namespace. The resolver must
detect conflicts between variants introduced by different enums in the same
module. Qualified access remains valid even when no conflict exists.

#### Unit erasure is total

The unit type `()` has no runtime representation. If it appears in fields,
variant payloads, parameters, arguments, pattern bindings, or function pointer
typedefs, it is erased at every one of those sites. Partial erasure is not
allowed.

## 9. Traits

Traits describe behavior. An impl may implement inherent methods, trait methods,
or both.

### 9.1 Trait declarations

Traits may declare methods, associated items, and supertraits.

### 9.2 Trait implementations

A concrete type implements a trait through an impl block that names both the
target type and the implemented trait.

### 9.3 Trait bounds

Generic parameters may be bounded by one or more traits.

### Implementation contracts

#### Bound-chain method lookup

When a receiver is a type parameter with trait bounds, method lookup must search:

1. the directly declared trait bounds
2. the supertraits of those bounds recursively

This is required so that a type parameter bounded by `Hashable` can resolve
methods declared on `Equatable` when `Hashable` extends `Equatable`.

#### Trait parameter ABI casts

When a trait type appears in a parameter position, the backend may need to cast
the concrete pointer type to the trait-representation pointer type at the call
site. Omitting this cast is a backend ABI bug.

## 10. Generics and monomorphization

### 10.1 Generic declarations

Functions, structs, enums, and traits may declare type parameters.

### 10.2 Type inference

Type arguments may be supplied explicitly or inferred from value arguments and
context.

### 10.3 Monomorphization model

The bootstrap compiler emits concrete specializations for generic uses. Generic
functions and generic types are not emitted directly to the bootstrap backend in
unresolved form.

### Implementation contracts

#### Valid specialization requires complete substitution

A specialization is valid if and only if every required type parameter has been
substituted with a concrete type. Required parameters include:

- the generic parameters declared by the function
- the generic parameters declared by its enclosing impl target, if any

Partial specialization must be rejected.

#### Recursive concreteness

A type is concrete only if it contains no unresolved type parameters anywhere in
its structure. Concreteness is recursive.

#### Specialization names include all type arguments

Mangled specialization names must distinguish all concrete type arguments.
`Option[ExprId]` and `Option[StmtId]` must not collide.

#### Only concrete generic instantiations are emitted

The backend emits only concrete instantiations. Unresolved generic types must be
filtered out before type emission.

#### Unresolved types are hard errors before codegen

If any MIR type reaching codegen is unresolved or unknown, the compiler must
emit a diagnostic and abort code generation. Substituting `Unknown` or `int` is
not allowed.

## 11. Modules and imports

### 11.1 Module model

Modules correspond to source files and directory structure according to the
repository layout.

### 11.2 Imports

Imports may refer to modules or to items within modules.

Examples:

```fuse
import core.list.List;
import util.math;
```

### 11.3 Re-exports and selective imports

The module system may re-export symbols and selectively expose imported items
according to the language surface.

### 11.4 Module loading

The standard library is implicitly available to every user module. Ordinary
Fuse code may refer to `String`, `List[T]`, `Option[T]`, `Result[T, E]`,
`Map[K, V]`, and other core types without any explicit import. Hosted
modules under `full/` and extended modules under `ext/` remain explicit —
a user that wants `io`, `chan`, or `net` must import them.

A user module may shadow a stdlib module by providing a module of the same
dotted path (for example, a user file mapped to module `core.string`). In
that case the user module wins and the stdlib copy is not loaded. Shadowing
is intended for testing and alternate implementations; ordinary programs
should not shadow stdlib modules.

The dependency direction across the stdlib tiers is one-way:

```
ext → full → core
```

`core` must not depend on `full` or `ext`, and `full` must not depend on
`ext`. This direction is enforced at resolve time, not just by convention.

### Implementation contracts

#### Module-first import resolution

Resolve an import by first treating the full dotted path as a module path. If no
module exists at that path, retry by treating the final segment as an item name
inside the preceding module path.

#### Stdlib modules are checked like user modules

Stdlib modules must be type-checked in the same pass as user modules. Any pass
that skips stdlib bodies while still lowering or codegening them violates the
frontend-backend contract.

#### Standard library is auto-loaded

Before parsing user sources, the driver loads the standard library
(`stdlib/core/` at minimum; `stdlib/full/` and `stdlib/ext/` on demand)
into the compilation unit. User sources merge on top: if a user source
declares a module whose path matches a stdlib module, the user version
takes precedence and the stdlib copy is omitted. If the stdlib root
cannot be located, the compiler emits a diagnostic and stops — there is
no silent fallback that compiles user code without stdlib types.

#### Nominal identity of generic instantiations follows the defining module

When user code in module `foo` writes `List[MyType]`, the specialization is
registered under the module where `List` is defined (`core.list`), not under
`foo`. Type identity for generic instantiations is `(defining_module, name,
type_args)`. A symbol-table miss during type resolution is a diagnostic, not
a silent fallback to the current module.

## 12. Concurrency

Fuse concurrency is explicit and structured.

### 12.1 Channels

`Chan[T]` is the primary message-passing primitive.

### 12.2 Shared state

`Shared[T]` represents shared mutable state protected by synchronization.

### 12.3 Lock ranking

`@rank(N)` is used to define static lock-ordering constraints. Missing ranks on
uses that require them are compile errors.

### 12.4 Threads

`spawn` creates concurrent execution explicitly. Fuse does not use `async` /
`await`.

## 13. Error handling

Fuse uses explicit result-based error handling.

### 13.1 Sum types

- `Option[T]` represents absence or presence
- `Result[T, E]` represents success or failure

### 13.2 Propagation

`?` propagates errors or absence according to the surrounding context.

### 13.3 No exceptions

Fuse does not define hidden exception flow as part of ordinary control flow.

## 14. The unsafe boundary

Unsafe operations are explicit.

### 14.1 Unsafe blocks

Operations that require `unsafe` include:

- dereferencing `Ptr[T]`
- raw pointer arithmetic
- calling FFI functions that cannot be proved safe
- unchecked indexing when the language surface exposes it

### 14.2 Visibility requirement

Unsafe operations must remain visible at their use site. Unsafe behavior must not
be smuggled through apparently safe APIs without explicit language support.

## 15. The C interop ABI

### 15.1 Extern declarations

FFI declarations use `extern` and name the symbol surface exposed to or imported
from the host environment.

### 15.2 Runtime bridge naming

Runtime bridge functions use a stable naming convention rooted in the runtime
surface, typically `fuse_rt_{module}_{operation}`.

### 15.3 Unsafe call-site rule

Every FFI call site is `unsafe` unless the language surface explicitly provides a
safe wrapper.

## 16. Backend representation contracts

This section defines the backend contracts that are not deducible solely from
surface syntax but are required for correct compiler implementation.

### 16.1 Contract 1: The two pointer categories

In the bootstrap C backend there are exactly two categories of pointer-typed
locals.

1. Borrow pointers.
2. `Ptr[T]` values.

Borrow pointers originate from borrow semantics and may need implicit
dereference to recover values. `Ptr[T]` values are first-class pointer values
and must never be implicitly dereferenced.

The backend must track these categories separately.

### 16.2 Contract 2: Unit erasure is total

Once `()` is erased in a concrete ABI position, it is erased at every producer
and consumer site. There is no partially materialized unit value in generated
code.

### 16.3 Contract 3: Monomorphization completeness

Every emitted generic specialization must be complete and concrete. The backend
must not emit calls to unresolved generic symbols.

### 16.4 Contract 4: Divergence is structural

A basic block ending in a diverging call has no successors. Code generation must
not synthesize reads of values that would exist only if control flow continued.

### 16.5 Contract 5: Emission order and aggregate fallback

Composite type definitions must be emitted before they are used by function
signatures or locals in generated C. Aggregate fallback values must be typed
zero-initializers such as `(FuseType_Foo){0}`, not scalar `0`.

### 16.6 Contract 6: Identifier sanitization and collision avoidance

All backend-emitted identifiers must be legal in the target backend language and
must be collision-resistant.

- C keywords must be escaped or renamed deterministically.
- numeric field names used internally must be sanitized
- same-named symbols from different modules must not collide after mangling

## 17. Grammar reference (EBNF)

The EBNF below defines the surface grammar at a reference level. Later
implementation documents may refine parser strategy, but they must not change the
language accepted by this grammar without updating this section.

```ebnf
file            = { import_decl | item_decl } ;

import_decl     = "import" path [ "as" IDENT ] ";" ;
path            = IDENT { "." IDENT } ;

item_decl       = fn_decl
								| struct_decl
								| enum_decl
								| trait_decl
								| impl_decl
								| const_decl
								| type_decl
								| extern_decl ;

fn_decl         = [ "pub" ] "fn" IDENT [ generic_params ]
									"(" [ param_list ] ")"
									[ "->" type_expr ]
									[ where_clause ]
									block_expr ;

param_list      = param { "," param } ;
param           = [ ownership ] IDENT ":" type_expr ;
ownership       = "ref" | "mutref" | "owned" ;

struct_decl     = [ "pub" ] [ decorator ] "struct" IDENT
									[ generic_params ]
									"{" [ field_list ] "}" ;

enum_decl       = [ "pub" ] "enum" IDENT [ generic_params ]
									"{" [ variant_list ] "}" ;

variant_list    = variant { "," variant } ;
variant         = IDENT
								| IDENT "(" [ type_list ] ")"
								| IDENT "{" [ field_list ] "}" ;

trait_decl      = [ "pub" ] "trait" IDENT [ generic_params ]
									[ ":" type_list ]
									"{" { trait_item } "}" ;

impl_decl       = "impl" [ generic_params ] type_expr
									[ ":" type_expr ]
									[ where_clause ]
									"{" { impl_item } "}" ;

type_expr       = path
								| tuple_type
								| array_type
								| slice_type
								| ptr_type
								| unit_type ;

expr            = assignment_expr ;
assignment_expr = logic_expr [ assign_op assignment_expr ] ;
logic_expr      = compare_expr { logic_op compare_expr } ;
compare_expr    = additive_expr { compare_op additive_expr } ;
additive_expr   = multiplicative_expr { add_op multiplicative_expr } ;
multiplicative_expr
								= unary_expr { mul_op unary_expr } ;
unary_expr      = [ unary_op ] postfix_expr ;
postfix_expr    = primary_expr { postfix_op } ;

primary_expr    = literal
								| path
								| block_expr
								| if_expr
								| match_expr
								| tuple_expr
								| struct_lit
								| "(" expr ")" ;

block_expr      = "{" { stmt } [ expr ] "}" ;
stmt            = let_stmt
								| var_stmt
								| return_stmt
								| break_stmt
								| continue_stmt
								| expr_stmt
								| item_decl ;
```