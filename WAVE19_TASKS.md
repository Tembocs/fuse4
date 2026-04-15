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

- [x] **8a.** token.fuse: define all TokenKind values as enum variants matching Stage 1
- [x] **8b.** token.fuse: define Token struct with kind, literal, span fields
- [x] **8c.** lexer.fuse: implement Lexer struct with source bytes, position, line/col tracking
- [x] **8d.** lexer.fuse: implement `next_token()` → scan whitespace, comments, identifiers
- [x] **8e.** lexer.fuse: implement keyword recognition from identifier text
- [x] **8f.** lexer.fuse: implement integer and float literal scanning with suffix detection
- [x] **8g.** lexer.fuse: implement string literal scanning with escape sequences
- [x] **8h.** lexer.fuse: implement operator and punctuation scanning
- [x] **8i.** E2e proof: lex a small Fuse program, verify token count and kinds

## 9. Stage 2 Compiler: Parser

- [x] **9a.** ast.fuse: define all AST node types matching Stage 1 (FnDecl, StructDecl, EnumDecl, etc.)
- [x] **9b.** ast.fuse: define all expression nodes (LiteralExpr, BinaryExpr, CallExpr, etc.)
- [x] **9c.** ast.fuse: define all statement and pattern nodes
- [x] **9d.** parser.fuse: implement Parser struct with token stream and position
- [x] **9e.** parser.fuse: implement item parsing (fn, struct, enum, trait, impl, const, type, extern, import)
- [x] **9f.** parser.fuse: implement expression parsing with Pratt precedence climbing
- [x] **9g.** parser.fuse: implement statement parsing (let, var, expr-stmt)
- [x] **9h.** parser.fuse: implement type expression parsing
- [x] **9i.** parser.fuse: implement pattern parsing for match arms
- [x] **9j.** parser.fuse: implement error recovery (synchronize to next statement/item)
- [x] **9k.** E2e proof: parse a small Fuse program, verify AST structure

## 10. Stage 2 Compiler: Name Resolution

- [x] **10a.** resolve.fuse: define Symbol, Scope, SymbolKind matching Stage 1
- [x] **10b.** resolve.fuse: define Module, ModulePath, ModuleGraph
- [x] **10c.** resolve.fuse: implement symbol indexing with enum variant hoisting
- [x] **10d.** resolve.fuse: implement import resolution with module-first fallback
- [x] **10e.** resolve.fuse: implement import cycle detection
- [x] **10f.** E2e proof: resolve symbols in a multi-module program

## 11. Stage 2 Compiler: Type System

- [x] **11a.** typetable.fuse: define TypeId, TypeKind, TypeEntry matching Stage 1
- [x] **11b.** typetable.fuse: implement type interning with dedup key
- [x] **11c.** typetable.fuse: implement all primitive type registration
- [x] **11d.** typetable.fuse: implement compound type constructors (Tuple, Array, Ref, Func, Struct, Enum)
- [x] **11e.** checker.fuse: implement two-pass checking (signatures then bodies)
- [x] **11f.** checker.fuse: implement expression type checking for all expression kinds
- [x] **11g.** checker.fuse: implement trait method lookup with supertrait chain
- [x] **11h.** checker.fuse: implement numeric widening and assignability
- [x] **11i.** E2e proof: type-check a program with structs, enums, generics, traits

## 12. Stage 2 Compiler: HIR and Liveness

- [x] **12a.** hir.fuse: define all HIR node types matching Stage 1
- [x] **12b.** hir.fuse: define Metadata with type, ownership, liveness fields
- [x] **12c.** hir.fuse: implement Builder with typed constructors for all node kinds
- [x] **12d.** Implement AST-to-HIR bridge in Stage 2 (matching driver/ast2hir.go)
- [x] **12e.** Implement liveness pass: compute LiveAfter and DestroyEnd
- [x] **12f.** E2e proof: build HIR for a function, verify metadata is populated

## 13. Stage 2 Compiler: MIR and Lowering

- [x] **13a.** mir.fuse: define Function, Block, Local, Instr, Terminator matching Stage 1
- [x] **13b.** mir.fuse: implement Builder with block/local/instruction emission
- [x] **13c.** Implement HIR-to-MIR lowering for all expression kinds
- [x] **13d.** Implement control flow lowering (if, match, while, loop, for, break, continue)
- [x] **13e.** Implement closure lowering (capture analysis, function lifting)
- [x] **13f.** E2e proof: lower a function to MIR, verify block structure

## 14. Stage 2 Compiler: Codegen

- [x] **14a.** codegen.fuse: implement C11 emitter matching Stage 1's emit.go
- [x] **14b.** codegen.fuse: implement type definition emission (structs, enums, tuples, arrays)
- [x] **14c.** codegen.fuse: implement function emission with instruction-by-instruction compilation
- [x] **14d.** codegen.fuse: implement name mangling matching Stage 1's mangle.go
- [x] **14e.** codegen.fuse: implement all 6 backend contracts (pointer categories, unit erasure, etc.)
- [x] **14f.** E2e proof: emit C source for a function, verify it compiles with gcc

## 15. Stage 2 Compiler: Driver and CLI

- [x] **15a.** driver.fuse: implement Build() orchestrating parse → resolve → check → lower → codegen
- [x] **15b.** driver.fuse: implement compileAndLink (write C, invoke gcc, link)
- [x] **15c.** main.fuse: implement CLI argument parsing (build, run, check subcommands)
- [x] **15d.** main.fuse: implement source file reading
- [x] **15e.** E2e proof: Stage 2 compiles a hello-world Fuse program

## 16. CLI Completeness

- [x] **16a.** Wire `fuse run`: after build, execute the binary and forward exit code
- [x] **16b.** Wire `fuse run`: forward stdout/stderr from child process
- [x] **16c.** Implement `fuse fmt`: read .fuse file, re-emit with consistent formatting
- [x] **16d.** Implement `fuse doc`: extract `///` doc comments from public items, emit markdown
- [x] **16e.** Implement `fuse test`: discover `*_test.fuse` files, compile and run, report pass/fail
- [x] **16f.** Implement `fuse repl`: read-eval-print loop with line-by-line compilation
- [x] **16g.** E2e proof: `fuse run hello.fuse` prints "hello" and exits 0
- [x] **16h.** E2e proof: `fuse check bad.fuse` reports error and exits 1

## 17. Native Backend (mandatory — required to retire C)

- [ ] **17a.** Implement x86-64 instruction encoding for all MIR instruction kinds
- [ ] **17b.** Implement register allocation (linear scan or graph coloring)
- [ ] **17c.** Implement System V calling convention (Linux/macOS)
- [ ] **17d.** Implement Win64 calling convention (Windows)
- [ ] **17e.** Implement ELF object file emission (Linux)
- [ ] **17f.** Implement PE/COFF object file emission (Windows)
- [ ] **17g.** Implement Mach-O object file emission (macOS)
- [ ] **17h.** Implement linker integration or bundled linker
- [ ] **17i.** E2e proof: compile and run hello-world without gcc on each platform
- [ ] **17j.** Verify all existing e2e tests pass through native backend
- [ ] **17k.** Remove C11 backend from the compiler path (retire gcc dependency)

## 18. Runtime Rewrite in Fuse (mandatory — required to retire C)

- [ ] **18a.** Rewrite runtime/src/mem.c in Fuse: alloc, realloc, free via platform syscalls
- [ ] **18b.** Rewrite runtime/src/panic.c in Fuse: panic, abort via platform syscalls
- [ ] **18c.** Rewrite runtime/src/io.c in Fuse: stdout, stderr, file I/O via platform syscalls
- [ ] **18d.** Rewrite runtime/src/proc.c in Fuse: argc, argv, env, exit via platform syscalls
- [ ] **18e.** Rewrite runtime/src/time.c in Fuse: monotonic clock via platform syscalls
- [ ] **18f.** Rewrite runtime/src/thread.c in Fuse: thread spawn via platform syscalls
- [ ] **18g.** Rewrite runtime/src/sync.c in Fuse: mutex, cond via platform syscalls
- [ ] **18h.** E2e proof: all runtime tests pass with Fuse runtime (no C runtime linked)
- [ ] **18i.** Remove C runtime source files from the build path

## 19. Self-Hosting Verification

- [ ] **19a.** Stage 1 compiles Stage 2 successfully (no errors)
- [ ] **19b.** Stage 2 compiles a test program correctly
- [ ] **19c.** Stage 2 compiles itself successfully
- [ ] **19d.** Compare Stage 1->Stage 2 output with Stage 2->Stage 2 output (reproducibility)
- [ ] **19e.** Add bootstrap health gate to CI (release-blocking)
- [ ] **19f.** Freeze Stage 1 as archival bootstrap tool
- [ ] **19g.** Remove Go from active build workflow
- [ ] **19h.** Remove C from active build workflow
- [ ] **19i.** Verify: `fuse build` produces working binaries with no Go, no C, no gcc
