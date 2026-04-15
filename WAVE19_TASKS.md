# Wave 19 Readiness — Complete Task Breakdown

## 0. Documentation

- [x] **0a.** Add L020 to learning-log.md documenting the full list of blockers between Wave 18 and Wave 19, why each is needed, and the dependency chain

## 1. Codegen: Pointer Write Support

- [x] **1a.** Add `InstrPtrWrite` to MIR instruction kinds: `dest_ptr[index] = value`
- [x] **1b.** Add `InstrPtrDerefWrite` to MIR instruction kinds: `*dest_ptr = value`
- [x] **1c.** Add `EmitPtrWrite(dest, index, value, elemType)` to MIR builder
- [x] **1d.** Add `EmitPtrDerefWrite(dest, value, elemType)` to MIR builder
- [x] **1e.** Emit C code for `InstrPtrWrite`: `dest[index] = value;` (with array .data support)
- [x] **1f.** Emit C code for `InstrPtrDerefWrite`: `*dest = value;`
- [x] **1g.** Add HIR node for pointer write expressions — handled via existing AssignExpr + IndexExpr
- [x] **1h.** Lowerer: lower assignment to indexed target (`arr[i] = v`) to `InstrPtrWrite`
- [x] **1i.** Lowerer: lower assignment to dereferenced pointer — handled via existing mutref deref
- [x] **1j.** Checker: type-check pointer index assignment — handled via existing assign + index checking
- [x] **1k.** Parser: parse `arr[index] = value` — already parsed as AssignExpr(IndexExpr, value)
- [x] **1l.** E2e proof: `arr[0] = 10; arr[1] = 32; arr[0] + arr[1]` exits 42

## 2. Codegen: String Concatenation

- [x] **2a.** Register `+` operator on String type in the checker (returns String)
- [x] **2b.** Lowerer: lower `String + String` to runtime call with out-params, construct String struct from results
- [x] **2c.** Runtime: add `fuse_rt_string_concat` in new `runtime/src/string.c` (out-param style)
- [x] **2d.** Runtime: add `fuse_rt_string_concat` declaration to `runtime/include/fuse_rt.h`
- [x] **2e.** Rebuild runtime library (`make runtime`)
- [x] **2f.** E2e proof: `"hello" + " world"` produces `"hello world"` in println output

## 3. Codegen: Self Type Resolution

- [x] **3a.** Checker: in `resolvePathType`, when name is `"Self"` and inside an impl block, resolve to the impl target type
- [x] **3b.** Track `currentImplTarget` TypeId in checker during impl body checking and signature registration
- [x] **3c.** Set `currentImplTarget` in `checkBodies` when entering an `ImplDecl` and in `registerImpl`
- [x] **3d.** Clear `currentImplTarget` after leaving the `ImplDecl` (defer)
- [x] **3e.** E2e proof: `impl Point { fn origin() -> Self }` compiles and returns correct value

## 4. Stdlib: List[T] Real Implementation

- [x] **4a.** List.push: if `len == cap`, compute new cap (cap * 2 or 8 if 0), call `fuse_rt_mem_realloc`, update data/cap
- [x] **4b.** List.push: write element at `data[len]` via pointer write, increment len
- [x] **4c.** List.get: bounds check `index < len`, read `data[index]` via pointer read, return `Some(elem)`
- [x] **4d.** List.pop: bounds check `len > 0`, decrement len, read `data[len]` via pointer read, return `Some(elem)`
- [x] **4e.** E2e proof: create List, push 3 values, get each, verify values match

## 5. Stdlib: Map[K, V] Real Implementation

- [x] **5a.** Define internal struct `MapEntry[K, V]` with key, value, occupied flag
- [x] **5b.** Map.new: allocate backing array of `MapEntry` with initial capacity
- [x] **5c.** Map.insert: hash key, linear probe for empty slot, write entry, increment len
- [x] **5d.** Map.get: hash key, linear probe for matching key, return `Some(value)` or `None`
- [x] **5e.** Map.remove: hash key, find entry, mark unoccupied, decrement len
- [x] **5f.** E2e proof: create Map, insert 3 entries, get each, verify values

## 6. Stdlib: Set[T] Real Implementation

- [x] **6a.** Set: change inner field from `len: USize` to `inner: Map[T, Bool]`
- [x] **6b.** Set.insert: delegate to `inner.insert(value, true)`
- [x] **6c.** Set.contains: delegate to `inner.get(ref value).is_some()`
- [x] **6d.** Set.remove: delegate to `inner.remove(ref value)`
- [x] **6e.** E2e proof: create Set, insert values, check contains, verify

## 7. Stdlib: String Operations

- [x] **7a.** String.toUpper: allocate new buffer, iterate bytes, convert 97-122 to 65-90, build new String
- [x] **7b.** String.toLower: allocate new buffer, iterate bytes, convert 65-90 to 97-122, build new String
- [x] **7c.** String equality by content: iterate both strings byte-by-byte, compare each
- [x] **7d.** String.contains: scan for substring match
- [x] **7e.** Formatter.write_str: concat string into buffer using string concat
- [x] **7f.** Formatter.write_char: append single char to buffer
- [x] **7g.** E2e proof: `"HELLO".toLower()` produces `"hello"`, string equality by content works

## 8. Stage 2 Compiler: Lexer

- [ ] **8a.** token.fuse: define all TokenKind values as enum variants matching Stage 1
- [ ] **8b.** token.fuse: define Token struct with kind, literal, span fields
- [ ] **8c.** lexer.fuse: implement Lexer struct with source bytes, position, line/col tracking
- [ ] **8d.** lexer.fuse: implement `next_token()` → scan whitespace, comments, identifiers
- [ ] **8e.** lexer.fuse: implement keyword recognition from identifier text
- [ ] **8f.** lexer.fuse: implement integer and float literal scanning with suffix detection
- [ ] **8g.** lexer.fuse: implement string literal scanning with escape sequences
- [ ] **8h.** lexer.fuse: implement operator and punctuation scanning
- [ ] **8i.** E2e proof: lex a small Fuse program, verify token count and kinds

## 9. Stage 2 Compiler: Parser

- [ ] **9a.** ast.fuse: define all AST node types matching Stage 1 (FnDecl, StructDecl, EnumDecl, etc.)
- [ ] **9b.** ast.fuse: define all expression nodes (LiteralExpr, BinaryExpr, CallExpr, etc.)
- [ ] **9c.** ast.fuse: define all statement and pattern nodes
- [ ] **9d.** parser.fuse: implement Parser struct with token stream and position
- [ ] **9e.** parser.fuse: implement item parsing (fn, struct, enum, trait, impl, const, type, extern, import)
- [ ] **9f.** parser.fuse: implement expression parsing with Pratt precedence climbing
- [ ] **9g.** parser.fuse: implement statement parsing (let, var, expr-stmt)
- [ ] **9h.** parser.fuse: implement type expression parsing
- [ ] **9i.** parser.fuse: implement pattern parsing for match arms
- [ ] **9j.** parser.fuse: implement error recovery (synchronize to next statement/item)
- [ ] **9k.** E2e proof: parse a small Fuse program, verify AST structure

## 10. Stage 2 Compiler: Name Resolution

- [ ] **10a.** resolve.fuse: define Symbol, Scope, SymbolKind matching Stage 1
- [ ] **10b.** resolve.fuse: define Module, ModulePath, ModuleGraph
- [ ] **10c.** resolve.fuse: implement symbol indexing with enum variant hoisting
- [ ] **10d.** resolve.fuse: implement import resolution with module-first fallback
- [ ] **10e.** resolve.fuse: implement import cycle detection
- [ ] **10f.** E2e proof: resolve symbols in a multi-module program

## 11. Stage 2 Compiler: Type System

- [ ] **11a.** typetable.fuse: define TypeId, TypeKind, TypeEntry matching Stage 1
- [ ] **11b.** typetable.fuse: implement type interning with dedup key
- [ ] **11c.** typetable.fuse: implement all primitive type registration
- [ ] **11d.** typetable.fuse: implement compound type constructors (Tuple, Array, Ref, Func, Struct, Enum)
- [ ] **11e.** checker.fuse: implement two-pass checking (signatures then bodies)
- [ ] **11f.** checker.fuse: implement expression type checking for all expression kinds
- [ ] **11g.** checker.fuse: implement trait method lookup with supertrait chain
- [ ] **11h.** checker.fuse: implement numeric widening and assignability
- [ ] **11i.** E2e proof: type-check a program with structs, enums, generics, traits

## 12. Stage 2 Compiler: HIR and Liveness

- [ ] **12a.** hir.fuse: define all HIR node types matching Stage 1
- [ ] **12b.** hir.fuse: define Metadata with type, ownership, liveness fields
- [ ] **12c.** hir.fuse: implement Builder with typed constructors for all node kinds
- [ ] **12d.** Implement AST-to-HIR bridge in Stage 2 (matching driver/ast2hir.go)
- [ ] **12e.** Implement liveness pass: compute LiveAfter and DestroyEnd
- [ ] **12f.** E2e proof: build HIR for a function, verify metadata is populated

## 13. Stage 2 Compiler: MIR and Lowering

- [ ] **13a.** mir.fuse: define Function, Block, Local, Instr, Terminator matching Stage 1
- [ ] **13b.** mir.fuse: implement Builder with block/local/instruction emission
- [ ] **13c.** Implement HIR-to-MIR lowering for all expression kinds
- [ ] **13d.** Implement control flow lowering (if, match, while, loop, for, break, continue)
- [ ] **13e.** Implement closure lowering (capture analysis, function lifting)
- [ ] **13f.** E2e proof: lower a function to MIR, verify block structure

## 14. Stage 2 Compiler: Codegen

- [ ] **14a.** codegen.fuse: implement C11 emitter matching Stage 1's emit.go
- [ ] **14b.** codegen.fuse: implement type definition emission (structs, enums, tuples, arrays)
- [ ] **14c.** codegen.fuse: implement function emission with instruction-by-instruction compilation
- [ ] **14d.** codegen.fuse: implement name mangling matching Stage 1's mangle.go
- [ ] **14e.** codegen.fuse: implement all 6 backend contracts (pointer categories, unit erasure, etc.)
- [ ] **14f.** E2e proof: emit C source for a function, verify it compiles with gcc

## 15. Stage 2 Compiler: Driver and CLI

- [ ] **15a.** driver.fuse: implement Build() orchestrating parse → resolve → check → lower → codegen
- [ ] **15b.** driver.fuse: implement compileAndLink (write C, invoke gcc, link)
- [ ] **15c.** main.fuse: implement CLI argument parsing (build, run, check subcommands)
- [ ] **15d.** main.fuse: implement source file reading
- [ ] **15e.** E2e proof: Stage 2 compiles a hello-world Fuse program

## 16. Native Backend (mandatory — required to retire C)

- [ ] **16a.** Implement x86-64 instruction encoding for all MIR instruction kinds
- [ ] **16b.** Implement register allocation (linear scan or graph coloring)
- [ ] **16c.** Implement System V calling convention (Linux/macOS)
- [ ] **16d.** Implement Win64 calling convention (Windows)
- [ ] **16e.** Implement ELF object file emission (Linux)
- [ ] **16f.** Implement PE/COFF object file emission (Windows)
- [ ] **16g.** Implement Mach-O object file emission (macOS)
- [ ] **16h.** Implement linker integration or bundled linker
- [ ] **16i.** E2e proof: compile and run hello-world without gcc on each platform
- [ ] **16j.** Verify all existing e2e tests pass through native backend
- [ ] **16k.** Remove C11 backend from the compiler path (retire gcc dependency)

## 17. Runtime Rewrite in Fuse (mandatory — required to retire C)

- [ ] **17a.** Rewrite runtime/src/mem.c in Fuse: alloc, realloc, free via platform syscalls
- [ ] **17b.** Rewrite runtime/src/panic.c in Fuse: panic, abort via platform syscalls
- [ ] **17c.** Rewrite runtime/src/io.c in Fuse: stdout, stderr, file I/O via platform syscalls
- [ ] **17d.** Rewrite runtime/src/proc.c in Fuse: argc, argv, env, exit via platform syscalls
- [ ] **17e.** Rewrite runtime/src/time.c in Fuse: monotonic clock via platform syscalls
- [ ] **17f.** Rewrite runtime/src/thread.c in Fuse: thread spawn via platform syscalls
- [ ] **17g.** Rewrite runtime/src/sync.c in Fuse: mutex, cond via platform syscalls
- [ ] **17h.** E2e proof: all runtime tests pass with Fuse runtime (no C runtime linked)
- [ ] **17i.** Remove C runtime source files from the build path

## 18. Self-Hosting Verification

- [ ] **18a.** Stage 1 compiles Stage 2 successfully (no errors)
- [ ] **18b.** Stage 2 compiles a test program correctly
- [ ] **18c.** Stage 2 compiles itself successfully
- [ ] **18d.** Compare Stage 1→Stage 2 output with Stage 2→Stage 2 output (reproducibility)
- [ ] **18e.** Add bootstrap health gate to CI (release-blocking)
- [ ] **18f.** Freeze Stage 1 as archival bootstrap tool
- [ ] **18g.** Remove Go from active build workflow
- [ ] **18h.** Remove C from active build workflow
- [ ] **18i.** Verify: `fuse build` produces working binaries with no Go, no C, no gcc
