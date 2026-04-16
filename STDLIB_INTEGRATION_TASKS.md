# Stdlib Integration — Refined Task Breakdown

The compiler cannot compile real programs that use the standard
library. See [learning-log L021](docs/learning-log.md) for the full
failure post-mortem. This document is the normative task breakdown
for the fix.

This breakdown was refined after cross-checking the original task
list against `docs/rules.md`, `docs/language-guide.md`,
`docs/implementation-plan.md`, `docs/repository-layout.md`, and
the current source. The changes versus the first draft are noted
inline in italics.

## Accountability

This work is accountable to the following rules. Every task must
respect them; tasks that cannot are rejected.

- **Rule 2.1** — The language guide precedes implementation.
  "Modules are auto-loaded" is a language decision, not a code
  decision. Section 0 updates the guide.
- **Rule 3.9** — Unresolved types must not reach codegen. Silent
  fallback to a phantom struct is forbidden. Sections 2 and 3
  replace silent fallbacks with diagnostics.
- **Rule 4.1 / 4.2** — Fix root causes, no workarounds. L021
  cataloged five band-aid fixes that were reverted. Sections 2–3
  target the three root causes, not the symptoms.
- **Rule 5.1** — Stdlib is the compiler's stress test. If a
  stdlib module fails to integrate, the compiler is wrong.
- **Rule 5.4** — `ext → full → core` dependency direction. The
  auto-load pass must not violate this.
- **Rule 6.8** — Every user-visible feature needs a proof program
  that compiles, links, runs, and produces verified output.
- **Rule 6.9** — Stubs emit diagnostics, not silent defaults.
- **Rule 7.2** — Mangling is stable and deterministic. Mangle
  changes in Section 4 are covered by golden tests.
- **Rule 9.1** — One logical change per commit. Each numbered
  section is one commit (or a short chain); sections do not mix.
- **Rule 10.3** — The learning log feeds the guide and plan.
  Section 0 updates both.

## Pre-implementation checklist (L014)

Before writing any production code in Sections 1–4:

- [x] **P1.** Write the Section 5 proof programs first, as Fuse
      source files (or inline `e2eTest` cases in
      `tests/e2e/e2e_test.go`). Do not stub them — use the shape
      the final code will have. *(landed in commit `60c5c2d`)*
- [x] **P2.** Run the e2e suite and capture the failure
      signatures. Commit the proof programs in a red state with
      the failure output pasted into the commit message.
      *(landed in commit `60c5c2d`; all 5 fail red, 77 prior
      subtests still green)*
- [x] **P3.** Only then begin Section 1. After each section,
      re-run the proofs; the failure signature should *change*,
      not vanish, until the relevant root cause is fixed.
      *(gate active; re-run after each section lands)*

*Refinement rationale:* The original doc listed proofs as
`Xd`/`Xv` bullets at the end of each section. Per L013/L014 that
is exactly the "self-verifying plan" pattern — proofs are written
after the fix and used to confirm, not to drive. Writing and
failing first is the discipline being re-applied here.

## 0. Documentation updates

L021 named a language-guide gap and a plan gap. Both must be
closed before implementation lands (Rule 2.1, Rule 10.3).

- [x] **0a.** `docs/learning-log.md` L021 landed in commit
      `ac60fc6`. *(already complete)*
- [x] **0b.** `docs/language-guide.md` — added §11.4 "Module
      loading" (user-facing) plus two new implementation
      contracts: "Standard library is auto-loaded" and "Nominal
      identity of generic instantiations follows the defining
      module." Covers (i) implicit stdlib availability,
      (ii) user shadowing, (iii) `ext → full → core` direction.
- [x] **0c.** `docs/implementation-plan.md` — added
      `[W18-P11-STDLIB-INTEGRATION]` with tasks T01–T06 covering
      auto-load, generic filter, module identity, secondary
      codegen fixes, proof programs, and band-aid regressions.
      Wave 18 exit criterion explicitly requires Phase 11 tasks
      to pass before Wave 19 starts.
- [x] **0d.** `docs/rules.md` — verified, no edit needed.
      Rule 3.9 (unresolved types), Rule 5.1 (stdlib is stress
      test), Rule 6.9 (no silent stubs) already cover the
      failure modes in L021.

*Refinement rationale:* Original doc had only task 0a. The
language-guide and plan updates are mandatory under Rule 2.1 and
Rule 10.3 and were missing.

## 1. Auto-load stdlib in `driver.Build()`

The driver currently compiles only the sources in `opts.Sources`.
User code referencing `String`, `List[T]`, `Result[T,E]` then
fails at codegen with incomplete C types.

### 1a. Implementation

- [x] **1a-i.** Implemented at the top of `driver.Build()` as
      Phase 0. The refined approach uses explicit opt-out
      (`SkipAutoStdlib bool`) rather than a heuristic over
      source keys — cleaner semantics and no false positives
      when a test uses a `core.math` key for its own fixture
      (observed in `TestBuildMultipleModules`).
- [x] **1a-ii.** User-wins merge implemented: stdlib entries
      are only added for keys absent from `opts.Sources`. The
      new e2e proof `stdlib_1b_ii_user_shadows_core_option`
      verifies a user `core.option` module takes precedence.
- [x] **1a-iii.** Missing stdlib root → diagnostic and stop.
      Empty `StdlibRoot()`, a path that doesn't exist, and a
      path that contains no `.fuse` files each emit a distinct
      error via `diagnostics.Errorf` and return early.
- [x] **1a-iv.** Rule 5.4 enforcement added as Phase 1.5.
      `stdlibTier()` + `tierAllows()` helpers in `driver.go`
      reject `core.*` imports of `full.*` or `ext.*` (and
      `full.*` imports of `ext.*`) with a concrete diagnostic.
- [x] **1a-v.** `BuildOptions.SkipAutoStdlib` added; existing
      callers that don't want auto-load opt out:
      `driver_test.go` (11 tests, now use `minimalBuild()`
      helper), `bootstrap.go` (already loads stdlib manually),
      `driver/stdlib_test.go`. New `BuildOptions.StdlibRoot`
      lets tests point at an explicit directory without
      relying on CWD or env.

### 1b. Proof

- [x] **1b-i.** `stdlib_1b_i_println_auto_load` — committed
      red. Failure signature shifted from "`println`
      unresolved / Fuse_main__String incomplete" to stdlib
      pipeline errors (`Fuse_core_bool__Formatter` unknown,
      method name collisions on `Fuse_eq`/`Fuse_ne`). Auto-
      load is confirmed working; remaining failures are
      Section 2 / 4a territory.
- [x] **1b-ii.** `stdlib_1b_ii_user_shadows_core_option` —
      committed red. Uses a user `core.option` module with
      a `UserOnly` variant that isn't in stdlib's version to
      prove the user version is what actually loaded.
      Currently fails for the same stdlib pipeline reasons
      as 1b-i; the shadow behavior itself works (no
      duplicate-symbol errors between user and stdlib
      `Option`).

**`TestCLIRunHello` is temporarily skipped** with a reference
to `W18-P11-T04`. It exercised a built-in `println` before
Section 1; now that the CLI auto-loads stdlib, the test fails
for the same stdlib pipeline reasons. It returns green when
Section 4 lands.

*Refinement rationale:* Original 1a–1d did not cover missing
stdlib root, tier direction enforcement, or the test harness
escape hatch. Without 1a-v, the existing
`TestStdlibCoreCompiles` would start pulling the whole stdlib
on every sub-test.

## 2. Filter generic templates in the codegen

Generic originals (`List[T]`, `Option[T]`, etc.) must never
reach C output. Only monomorphized specializations
(`List[I32]`, `Option[String]`) produce types and functions.

### 2a. Reuse `monomorph.IsGeneric`

- [ ] **2a-i.** **Do not** add a new `hasGenericParam` helper.
      `compiler/monomorph/monomorph.go:64-75` already has
      `Context.IsGeneric(TypeId) bool` that walks types
      recursively. Expose it as a package-level function or
      pass a `*monomorph.Context` into the emitter.
- [ ] **2a-ii.** If importing monomorph into codegen creates a
      cycle, move `IsGeneric` to `compiler/typetable/` as
      `(*TypeTable).HasGenericParam(TypeId) bool` — it only
      depends on TypeTable state anyway.

*Refinement rationale:* Original 2a added a duplicate
`hasGenericParam`. Duplicate definitions of "is this type
generic" are a recipe for drift (one says yes, the other says
no, and codegen emits half a specialization). Reuse the
existing implementation.

### 2b. Skip generic types in `emitTypeDefIfNeeded`

- [ ] **2b-i.** At the top of `emitTypeDefIfNeeded`
      (`compiler/codegen/emit.go:82`): if the type is
      `KindGenericParam`, mark emitted and return.
- [ ] **2b-ii.** For `KindStruct` and `KindEnum`, if
      `HasGenericParam(id)` is true, mark emitted and return.
- [ ] **2b-iii.** Verification: add a test that parses a program
      using `List[I32]`, runs the full pipeline, and asserts via
      regex that the generated C output contains no
      `Fuse_*__T(?![0-9A-Za-z_])` (base generic param name)
      identifiers.

### 2c. Skip generic functions in the emitter

- [ ] **2c-i.** Add `(*typetable.TypeTable).FnHasGenericParam(*mir.Function) bool`
      or place it in codegen — whichever avoids the cycle. Walks
      params, locals, and return type.
- [ ] **2c-ii.** In `collectTypes`, skip the function if
      `FnHasGenericParam` returns true.
- [ ] **2c-iii.** In `emitFnForwardDecl` and `emitFunction`,
      same skip.
- [ ] **2c-iv.** Guard: if the driver correctly filters generic
      originals (already done in `driver.go:92-96`), this filter
      catches nothing in practice but defends against
      regressions. Add a test that inserts a generic MIR
      function into the emitter and asserts it is skipped.

### 2d. Verification

- [ ] **2d-i.** `go test ./compiler/...` — green.
- [ ] **2d-ii.** `go test -run TestE2E ./tests/e2e/` — green.
- [ ] **2d-iii.** `go test -run TestStdlib ./tests/e2e/` — green.
- [ ] **2d-iv.** `go test -run TestStage1CompilesStage2 ./compiler/driver/` — green.
- [ ] **2d-v.** `go vet ./...` — clean.

*Refinement rationale:* Original 2a duplicated a helper. 2c-iv
adds a regression-style test so a future driver change that
stops filtering generic originals doesn't silently re-introduce
the bug.

## 3. Canonicalize module identity for generic instantiations

When user code in module `foo` references `List[MyType]`, the
specialization must register under `core.list`, not `foo`.

### 3a. Root-cause fix in `resolvePathType`

- [ ] **3a-i.** `compiler/check/types.go:43-87` — when the
      symbol table lookup at lines 68-79 succeeds, the code
      already uses `sym.Module.String()`. Verify this path
      triggers for auto-loaded stdlib types (it should, once
      Section 1 lands and `List`, `Option`, etc. are in the
      symbol table).
- [ ] **3a-ii.** Replace the lines 82-86 fallback
      (`InternStruct(modStr, name, typeArgs)` under the current
      module) with an **error diagnostic**:
      `"unresolved type '<name>'"`. This satisfies Rule 6.9 and
      Rule 3.9. The only legal fallthrough to a synthetic
      TypeId is when `name` is a generic parameter of the
      currently-checked function (handled elsewhere — verify).
- [ ] **3a-iii.** Add a unit test: a program referencing a type
      name that exists in no module produces a diagnostic,
      not a silent phantom struct.

*Refinement rationale:* Original 3a-ii added a `coreTypeLookup`
hard-coded table as a fallback. That re-introduces the silent
phantom-struct pattern one layer down. Once Section 1 makes
stdlib symbols actually visible, the existing symbol-table
lookup at 68-79 works unchanged. The **correct** fallback when
a name is unknown is a diagnostic, not a hard-coded core
lookup. The `coreTypeLookup` idea must be removed.

### 3b. Same root-cause fix in `checkStructLit`

- [ ] **3b-i.** `compiler/check/expr.go:725-734` — replace the
      unconditional `InternStruct(modStr, e.Name, nil)` with a
      symbol-table lookup mirroring `resolvePathType`. If the
      name resolves to a known struct symbol, use that symbol's
      module. If it resolves to nothing, emit
      `"unknown struct '<name>'"`.
- [ ] **3b-ii.** Share the lookup path between the two sites
      via a helper `(*Checker).resolveTypeName(name string) (modStr string, ok bool)`
      to avoid drift.

### 3c. Emit specialized struct/enum definitions

Once 3a and 3b are correct, `List[I32]` is interned with
module `core.list`, and `List[T]` (the base) is already there
too. The emitter needs to substitute.

- [ ] **3c-i.** In `emitTypeDefIfNeeded` for `KindStruct`,
      when the type has `TypeArgs` but empty `Fields` (i.e.,
      the specialization was interned without a layout), look
      up the base: the TypeTable's `InternStruct` returns the
      existing entry for a given (module, name) with nil
      TypeArgs, so the base is always findable by module+name.
- [ ] **3c-ii.** Add `(*TypeTable).BaseOf(id TypeId) TypeId`
      — returns the TypeId of the generic template (same
      module, same name, nil typeArgs). Returns `InvalidTypeId`
      if the input is not a specialization.
- [ ] **3c-iii.** Add `(*TypeTable).SubstituteFields(baseId TypeId, typeArgs []TypeId) ([]string, []TypeId)`
      — walks the base type's fields, replaces
      `KindGenericParam` references (matched by `Name`) with the
      corresponding TypeArg, and recurses into `KindPtr`,
      `KindRef`, `KindMutRef`, `KindSlice`, `KindArray`,
      `KindTuple`, `KindStruct`, `KindEnum`. Reuse the
      substitution logic from `monomorph.Context.Substitute`
      rather than duplicate it.
- [ ] **3c-iv.** Apply the same pattern for `KindEnum`.
- [ ] **3c-v.** **Do not** add a "try canonical core module if
      BaseOf fails" fallback. If 3a is correct, BaseOf finds
      the base. Adding a fallback re-introduces the root-cause
      bug in disguise.

*Refinement rationale:* Original 3c-v was a symptom patch
making the "module identity mismatch" silently recoverable.
Remove it; rely on Section 3a's root-cause fix.

### 3d. Register base fields for generic enums

- [ ] **3d-i.** `compiler/check/checker.go:214-256` — in
      `registerEnum`, for generic enums (those with
      `GenericParams`), also intern the base type
      (`InternEnum(modStr, name, nil)`) and call `SetEnumFields`
      on it with the variant payload types. Currently only
      `EnumVariants[name]` is populated for generic enums;
      `SetEnumFields` is only called for non-generic enums at
      line 254. `SubstituteFields` needs the base to have
      fields to substitute.
- [ ] **3d-ii.** Proof program: a function returning
      `Result[(), String]` compiles. Generated C has a tagged
      union with `_tag`, `_f0` (unit-erased), and `_f1`
      (String struct).

## 4. Secondary codegen issues

These become visible once Sections 1–3 are in. Each is a small
root-cause fix; none depends on the others.

### 4a. Qualify trait/impl method names with target type

- [ ] **4a-i.** Context: `registerSpecializedImplMethods`
      (`checker.go:371-395`) already registers methods for
      monomorphized impls under `baseName.method`. The missing
      piece is *non-generic* trait impls: if two types both
      implement `Equatable`, both emit `Fuse_eq` and collide.
- [ ] **4a-ii.** In `compiler/driver/driver.go` inside the
      `ImplDecl` branch (currently lines 103-119), rename each
      impl method's HIR function to
      `{TargetTypeName}__{method}` before lowering.
- [ ] **4a-iii.** In `compiler/lower/lower.go`, in method call
      lowering, qualify the callee name with the receiver type
      (same scheme). This is where method-dispatch names are
      decided, not in codegen — Rule 3.1 (three IRs, no
      skipping).
- [ ] **4a-iv.** Trait default methods inherited by empty impls
      already get cloned by `registerImpl` at
      `checker.go:499-531`. Verify the clone inherits the
      qualified name scheme.
- [ ] **4a-v.** Drop call site — `emit.go:447` currently emits
      `Fuse_drop(...)`. Change to `<TypeName>_drop(...)` using
      the same scheme. *(Already implemented at line 313 for
      `InstrDrop`; line 447 is the end-of-function path and
      uses the generic name.)*
- [ ] **4a-vi.** Proof: two types (`I32` and `String`) both
      implementing `Equatable` link without duplicate-symbol
      errors; both `eq` calls return correct results.

### 4b. Preserve extern function names

- [ ] **4b-i.** `compiler/codegen/mangle.go:51-59` —
      `MangleName`: if `name` starts with `fuse_rt_`, return it
      unchanged, no `Fuse_` prefix.
- [ ] **4b-ii.** Coordinate with 4a: the method-qualification
      scheme must not prefix extern names either. Single
      guard at the top of `MangleName` handles both.
- [ ] **4b-iii.** `compiler/lower/lower.go`, direct-call path
      — if the identifier name starts with `fuse_rt_`, do not
      build a mangled callee through `MangleName` at all;
      leave the raw name.
- [ ] **4b-iv.** Mangle golden: add a test in
      `compiler/codegen/codegen_test.go` that fixes the exact
      mangled output for each class (`Fuse_add`, `main`,
      `fuse_rt_proc_argc`, `I32__eq`). Rule 7.2.

*Refinement rationale:* Original said 4a–4d were "independent"
but 4a and 4b both edit `mangle.go`'s `MangleName`. They must
be coordinated or one clobbers the other. 4b-iv adds the
golden test mandated by Rule 7.2.

### 4c. Strip numeric literal suffixes

- [ ] **4c-i.** `compiler/codegen/emit.go:533` `constValue` —
      before returning a numeric value, call a
      `stripNumericSuffix(s)` helper that removes trailing
      `usize|isize|u128|u64|u32|u16|u8|i128|i64|i32|i16|i8|f64|f32`
      if the remainder parses as a number.
- [ ] **4c-ii.** Precedence: strip longest suffix first
      (`usize` before `u`).
- [ ] **4c-iii.** Proof: `let x = 0usize; let y = 42u8;`
      generates C containing `0` and `42`, not `0usize`/`42u8`.

### 4d. Borrow-of-borrow double-pointer — fix in the lowerer

- [ ] **4d-i.** Root cause is in `compiler/lower/lower.go`,
      not the emitter. When lowering a `ref x` or `mutref x`
      where `x` already has type `KindRef` / `KindMutRef`
      (because it came from a param), the lowerer should emit
      an `InstrCopy`, not an `InstrBorrow`. The reference is
      already a pointer; taking its address would be
      `String**`.
- [ ] **4d-ii.** Do **not** add the `codegen/emit.go` fix
      proposed in the original 4d. That covers the symptom
      (already-pointer locals getting `&` prefixed) but
      conflicts with Rule 3.1 / Rule 4.1: correct borrow
      semantics belong in HIR→MIR lowering, not in a C-string
      patch.
- [ ] **4d-iii.** Proof: a function taking `mutref String`
      and passing it to another function compiles. The
      generated C has matching pointer levels and no
      `incompatible pointer type` warnings from gcc
      (`-Wincompatible-pointer-types` should be clean).

*Refinement rationale:* Original 4d patched the codegen. That
is Rule 4.2 (workarounds forbidden). The correct place to
decide "this is already a borrow, do not re-borrow" is the
lowerer.

## 5. Proof programs

These programs are the exit criteria. Each must compile, link,
run, and produce the expected exit code or stdout.

- [ ] **5a.** Struct with String fields:
      `struct Config { name: String, version: String } ...`
- [ ] **5b.** Struct with generic collection:
      `struct Registry { entries: List[Entry] } ...` uses
      `push`, `get`, `len`.
- [ ] **5c.** `Result[String, String]` with `?` operator and
      `match` on Ok/Err.
- [ ] **5d.** `String.contains`, `String.starts_with`,
      `String.byte_at`.
- [ ] **5e.** `Map[String, I32]` with `insert`, `get`, `len`.
- [ ] **5f.** `python test_all.py` — all 7 steps pass.

Each of 5a–5e is an e2e test case in
`tests/e2e/e2e_test.go` or a new `tests/e2e/real_programs_test.go`.

## 6. Regression coverage for the 18-commit band-aid spiral

L021 catalogs five approaches that were tried and reverted.
Per Rule 4.3, each distinct failure mode must become a
regression test so a future change cannot reintroduce it.

- [ ] **6a.** "Hardcoded String typedef pre-emitted" — test
      that the generated C does not contain any hand-written
      `typedef struct Fuse_core_String_...` outside of what
      `emitTypeDefIfNeeded` produces. A grep-based assertion
      on generated C.
- [ ] **6b.** "`coreTypeLookup` table consulted by emitter" —
      test that no such table exists in the codegen package
      (e.g., a `go vet`-like check that the codegen package
      has no map from type name to canonical module).
- [ ] **6c.** "Generic function signature leaks `T` into C" —
      covered by Section 2's regex test (2b-iii), make it
      explicit.
- [ ] **6d.** "Generic struct typedef leaks `T`" — same.
- [ ] **6e.** "`FindBaseType` silently falls back to canonical
      core module" — test that with 3a correct, `BaseOf`
      returns a valid id without any fallback branch being
      reached.

*Refinement rationale:* Original doc had no regression
coverage of the failure spiral. Without these, a future well-
meaning change can re-spiral.

## 7. Implementation order and commit plan

```
Section 0 (docs)       — 1 commit, 4 small edits
    ↓
Pre-check (P1–P3)      — 1 commit: failing proofs captured in red
    ↓
Section 1 (auto-load)  — 1 commit
    ↓
Section 2 (generic filter)  — 1 commit
    ↓
Section 3 (module id)  — 1 commit (3a+3b together, 3c+3d together if small)
    ↓
Section 4a / 4b / 4c / 4d — 4 commits (independent; 4a+4b coordinated in mangle.go)
    ↓
Section 5 (proofs flip green) — 1 commit
    ↓
Section 6 (regressions) — 1 commit
```

Rule 9.1: one logical change per commit. Total: ~10 commits.

## 8. Files affected (verified against current code)

| File | Role | Sections |
|------|------|----------|
| `docs/language-guide.md` | Module-loading spec | 0b |
| `docs/implementation-plan.md` | Schedule this work | 0c |
| `compiler/driver/driver.go` | Auto-load, trait-impl method rename | 1, 4a |
| `compiler/driver/stdlib.go` | `LoadStdlib` (already exists, unchanged) | 1 (caller) |
| `compiler/check/types.go` | `resolvePathType` diagnostic | 3a |
| `compiler/check/expr.go` | `checkStructLit` symbol lookup | 3b |
| `compiler/check/checker.go` | Base-enum field registration | 3d |
| `compiler/typetable/typetable.go` | `HasGenericParam`, `BaseOf`, `SubstituteFields` | 2a-ii, 3c-ii, 3c-iii |
| `compiler/codegen/emit.go` | Generic filter, specialization emission, numeric suffix | 2b, 2c, 3c, 4c |
| `compiler/codegen/mangle.go` | Extern passthrough, method prefix | 4a, 4b |
| `compiler/lower/lower.go` | Method-name qualification, extern passthrough, borrow-of-borrow | 4a, 4b, 4d |
| `tests/e2e/e2e_test.go` | Proofs 5a–5e | 5 |
| `tests/e2e/stdlib_test.go` | Use `SkipAutoStdlib` escape hatch | 1a-v |
| `compiler/codegen/codegen_test.go` | Mangle goldens | 4b-iv |

## 9. Totals

- ~50 tasks across 7 sections.
- Documentation updates (Section 0) and pre-implementation red
  state (P1–P3) are now explicit.
- Five band-aid regressions (Section 6) are new.
- The three "fallback to canonical core" patterns from the
  original draft have been removed and replaced with
  diagnostics (Sections 2, 3).

If implementation discovers a root cause not named here, stop
and update this document rather than adding another fallback.
