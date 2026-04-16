# Stdlib Integration — Complete Task Breakdown

The compiler cannot compile real programs that use the standard library.
This document breaks down the fix into granular, testable tasks.

See learning-log L021 for the full diagnosis.

## 0. Documentation

- [ ] **0a.** Add L021 to learning-log.md documenting the failure, root causes, and architectural lesson

## 1. Auto-load stdlib in `driver.Build()`

- [ ] **1a.** In `compiler/driver/driver.go`, at the top of `Build()`, check if `opts.Sources` already contains `core.string` — if not, load stdlib
- [ ] **1b.** Call `LoadStdlib(StdlibRoot())` to get all stdlib source files
- [ ] **1c.** Merge stdlib sources into `opts.Sources`, skipping keys that already exist (user sources take precedence)
- [ ] **1d.** E2e proof: `fn main() -> I32 { println("hello"); return 0; }` compiles and runs (stdlib auto-loaded, not passed on command line)

## 2. Filter generic templates in the codegen

Generic originals (e.g., `List[T]`, `Option[T]`, `Result[T, E]`) must
not produce C output. Only their monomorphized specializations (e.g.,
`List[I32]`, `Option[String]`) should be emitted.

### 2a. Add `hasGenericParam` helper to emitter

- [ ] **2a-i.** Add `func (e *Emitter) hasGenericParam(id TypeId) bool` to `compiler/codegen/emit.go`
- [ ] **2a-ii.** The function walks a type recursively: returns true if the type itself is `KindGenericParam`, or if any of its `Fields` or `Elem` references a `KindGenericParam`
- [ ] **2a-iii.** Unit test: `hasGenericParam` returns true for `Ptr[T]` where `T` is `KindGenericParam`, false for `Ptr[I32]`

### 2b. Skip generic types in `emitTypeDefIfNeeded`

- [ ] **2b-i.** At the top of `emitTypeDefIfNeeded`, if the type is `KindGenericParam`, mark emitted and return
- [ ] **2b-ii.** For `KindStruct` and `KindEnum`, if `hasGenericParam(id)` is true, mark emitted and return (do not emit a C typedef)
- [ ] **2b-iii.** Verify: after this change, no `Fuse_core_list__T` or `Fuse_core_option__T` appears in any generated C output

### 2c. Skip generic functions in the emitter

- [ ] **2c-i.** Add `func (e *Emitter) fnHasGenericTypes(fn *mir.Function) bool` — returns true if any param, local, or return type has a generic param
- [ ] **2c-ii.** In `collectTypes`, skip the function if `fnHasGenericTypes` returns true
- [ ] **2c-iii.** In `emitFnForwardDecl`, skip the function if `fnHasGenericTypes` returns true
- [ ] **2c-iv.** In `emitFunction`, skip the function if `fnHasGenericTypes` returns true
- [ ] **2c-v.** E2e proof: `fn main() -> I32 { return 0; }` compiles with stdlib auto-loaded — no generic type errors in gcc output

### 2d. Verify existing tests still pass

- [ ] **2d-i.** `go test ./compiler/...` — all compiler unit tests pass
- [ ] **2d-ii.** `go test -run TestE2E$ ./tests/e2e/` — all e2e tests pass
- [ ] **2d-iii.** `go test -run TestStdlib ./tests/e2e/` — all stdlib compilation tests pass
- [ ] **2d-iv.** `go test -run TestStage1CompilesStage2 ./compiler/driver/` — bootstrap passes

## 3. Canonicalize module identity for generic instantiations

When user code in module `foo` references `List[MyType]`, the checker
must register the instantiation under `core.list` (where `List` is
defined), not under `foo`.

### 3a. Fix `resolvePathType` in the checker

- [ ] **3a-i.** In `compiler/check/types.go`, in `resolvePathType`, when the symbol table lookup at line 68-79 finds the symbol, use `sym.Module.String()` as the module for `InternStruct` / `InternEnum` — this is already the case, verify it works with auto-loaded stdlib
- [ ] **3a-ii.** When the symbol table lookup FAILS (line 86 fallback), check if the name is a well-known core type (`String`, `List`, `Map`, `Set`, `Option`, `Result`, `Hasher`, `Formatter`) and use its canonical module path instead of the current module
- [ ] **3a-iii.** Add a `coreTypeLookup(name string) string` function that maps type names to canonical modules: `String` → `core.string`, `List` → `core.list`, `Option` → `core.option`, `Result` → `core.result`, etc.
- [ ] **3a-iv.** Guard the core type lookup: only apply when the current module is NOT the defining module itself (prevents mismatch when compiling `core/string.fuse` where the current module IS `core.string`)
- [ ] **3a-v.** Add `isDefiningModule(currentMod, coreModule string) bool` that checks exact match and suffix match (e.g., `"string"` matches `"core.string"` for the test harness)

### 3b. Fix `checkStructLit` in the checker

- [ ] **3b-i.** In `compiler/check/expr.go`, in `checkStructLit`, apply the same `coreTypeLookup` before falling back to the current module for the struct literal's type
- [ ] **3b-ii.** Apply the same `isDefiningModule` guard

### 3c. Emit specialized generic struct/enum definitions

- [ ] **3c-i.** In `emitTypeDefIfNeeded` for `KindStruct`, when a type has `TypeArgs` but no `Fields` (specialized instantiation without field layout), look up the base type using `FindBaseType(kind, module, name)`
- [ ] **3c-ii.** Add `FindBaseType(kind TypeKind, module, name string) TypeId` to `compiler/typetable/typetable.go` — looks up the unspecialized type by intern key
- [ ] **3c-iii.** Add `SubstituteFields(baseId TypeId, typeArgs []TypeId) ([]string, []TypeId)` to `compiler/typetable/typetable.go` — walks the base type's fields, replaces `KindGenericParam` references with corresponding TypeArgs
- [ ] **3c-iv.** The substitution must handle `Ptr[T]` → `Ptr[ConcreteType]` by recursing into pointer element types
- [ ] **3c-v.** If `FindBaseType` fails under the instantiation's module, try the canonical core module via `coreModuleForType` (same table as `coreTypeLookup` but in the codegen package)
- [ ] **3c-vi.** Apply the same pattern for `KindEnum` specializations
- [ ] **3c-vii.** E2e proof: `struct Bag { items: List[I32] }` compiles — the generated C has a full typedef for `List[I32]` with `int32_t* data; uintptr_t len; uintptr_t cap;`

### 3d. Register base fields for generic enums

- [ ] **3d-i.** In `compiler/check/checker.go`, in `registerEnum`, for generic enums (those with `GenericParams`), also call `SetEnumFields` on the base type with the variant payload types — this enables `SubstituteFields` to work for `Result[T, E]` and `Option[T]`
- [ ] **3d-ii.** E2e proof: a function returning `Result[(), String]` compiles — the generated C has a full tagged-union typedef

## 4. Fix secondary codegen issues

These are downstream issues that become visible once sections 1-3 are
working.

### 4a. Qualify trait/impl method names with target type

- [ ] **4a-i.** In `compiler/driver/driver.go`, for `ImplDecl` items, extract the target type name from `impl.Target`
- [ ] **4a-ii.** After building the HIR function for an impl method, rename it to `{TargetType}__{method}` (e.g., `I32__eq`, `String__eq`, `Counter__get`)
- [ ] **4a-iii.** In `compiler/lower/lower.go`, in the method call path, qualify the callee name with the receiver type: `Fuse_{TypeName}__{method}`
- [ ] **4a-iv.** For trait default methods inherited by empty impl blocks, emit a copy of the function qualified with each implementing type's name
- [ ] **4a-v.** Update the drop destructor call in `emitTerminator` to use `Fuse_{TypeName}__drop` instead of `Fuse_drop`
- [ ] **4a-vi.** E2e proof: two types implementing `Equatable` compile — both `I32__eq` and `String__eq` are callable

### 4b. Preserve extern function names

- [ ] **4b-i.** In `compiler/codegen/mangle.go`, in `MangleName`, if the name starts with `fuse_rt_`, return it as-is without the `Fuse_` prefix
- [ ] **4b-ii.** In `compiler/lower/lower.go`, in the direct call path, if the identifier name starts with `fuse_rt_`, do not prepend `Fuse_`
- [ ] **4b-iii.** E2e proof: `unsafe { extern fn fuse_rt_proc_argc() -> I32; fuse_rt_proc_argc(); }` compiles — the C output calls `fuse_rt_proc_argc`, not `Fuse_fuse_rt_proc_argc`

### 4c. Strip numeric literal suffixes

- [ ] **4c-i.** In `compiler/codegen/emit.go`, in `constValue`, before returning the raw value, strip Fuse numeric suffixes: `usize`, `isize`, `u64`, `i64`, `u32`, `i32`, `u16`, `i16`, `u8`, `i8`, `f64`, `f32`
- [ ] **4c-ii.** Add a `stripNumericSuffix(s string) string` helper that checks for each suffix and strips it if the remaining prefix is a valid number
- [ ] **4c-iii.** E2e proof: `let x = 0usize; let y = 42u8;` compiles — the C output has `0` and `42`, not `0usize` and `42u8`

### 4d. Avoid double-pointer on borrow of borrow

- [ ] **4d-i.** In `compiler/codegen/emit.go`, in `emitInstr` for `InstrBorrow`, check if the source local's type is already `KindRef` or `KindMutRef`
- [ ] **4d-ii.** If yes, emit a direct assignment (`dest = src;`) instead of `dest = &src;` — this avoids `String**` when `String*` was expected
- [ ] **4d-iii.** E2e proof: a function taking `mutref String` and passing it to another function compiles — no `incompatible pointer type` warnings

## 5. End-to-end proof programs

These prove that a real multi-module program can be compiled.

- [ ] **5a.** E2e proof: program with `struct Config { name: String, version: String }` — compiles, constructs, accesses fields, prints
- [ ] **5b.** E2e proof: program with `struct Registry { entries: List[Entry] }` — uses push, get, len
- [ ] **5c.** E2e proof: program with function returning `Result[String, String]` — uses `?` operator, match on Ok/Err
- [ ] **5d.** E2e proof: program calling `String.contains`, `String.starts_with`, `String.byte_at` — stdlib method calls work
- [ ] **5e.** E2e proof: program with `Map[String, I32]` — insert, get, len
- [ ] **5f.** `python test_all.py` — all 7 steps pass after all changes

## Implementation order

```
Section 1 (auto-load stdlib)
    ↓
Section 2 (filter generics in emitter)
    ↓
Section 3 (canonicalize module identity)
    ↓
Section 4a (method name qualification)  \
Section 4b (extern name preservation)    } independent, can be done in any order
Section 4c (numeric suffix stripping)   /
Section 4d (double-pointer fix)        /
    ↓
Section 5 (end-to-end proofs)
```

Sections 1-3 are the critical path. If those are correct, sections 4a-4d
are mechanical fixes. Section 5 is verification only.

## Files affected

| File | Changes |
|------|---------|
| `compiler/driver/driver.go` | Auto-load stdlib, method name qualification, trait default forwarding |
| `compiler/codegen/emit.go` | Generic type filter, generic function filter, numeric suffix strip, double-pointer fix, drop name fix |
| `compiler/codegen/mangle.go` | Extern name preservation |
| `compiler/check/types.go` | `coreTypeLookup`, `isDefiningModule` |
| `compiler/check/expr.go` | Struct literal core type lookup |
| `compiler/check/checker.go` | Generic enum base field registration |
| `compiler/typetable/typetable.go` | `FindBaseType`, `SubstituteFields` |
| `compiler/lower/lower.go` | Method name qualification, extern name preservation |
| `tests/e2e/e2e_test.go` | New proof programs (5a-5e) |

## Total: 42 tasks across 5 sections
