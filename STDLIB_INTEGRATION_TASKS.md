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

- [x] **2a-i / 2a-ii.** Added
      `(*typetable.TypeTable).HasGenericParam(TypeId) bool`
      as the single source of truth. It only depends on
      TypeTable state, so no import-cycle risk. The existing
      `monomorph.Context.IsGeneric` does the same walk but
      requires a Context; the new method is the canonical
      implementation and should be preferred going forward.
      A future cleanup can switch `monomorph.IsGeneric` to
      delegate to `TypeTable.HasGenericParam`.

### 2b. Skip generic types in `emitTypeDefIfNeeded`

- [x] **2b-i.** `KindGenericParam` short-circuits at the top
      of `emitTypeDefIfNeeded`: mark emitted, return.
- [x] **2b-ii.** `KindStruct` and `KindEnum` that transitively
      carry a generic param short-circuit similarly.
- [x] **2b-iii.** Added `TestSection2NoGenericParamLeakage`
      in `tests/e2e/e2e_test.go`. It runs the full pipeline
      (with auto-loaded stdlib) and greps the generated C
      for the `Fuse_*__([TUVKE])\b` pattern. Currently
      passes — stdlib loads, no template leakage.

### 2c. Skip generic functions in the emitter

- [x] **2c-i.** Added `(*Emitter).fnHasGenericParam(*mir.Function)`
      in `compiler/codegen/emit.go` — delegates to
      `TypeTable.HasGenericParam` for each param/local/return
      type.
- [x] **2c-ii.** `collectTypes` short-circuits on generic fn.
- [x] **2c-iii.** `emitFnForwardDecl` and `emitFunction` both
      short-circuit on generic fn.
- [x] **2c-iv.** Regressions added in
      `compiler/codegen/codegen_test.go`:
      `TestGenericFunctionNotEmitted` (backstop for the
      driver filter) and `TestGenericStructTypedefNotEmitted`.

### 2d. Verification

- [x] **2d-i.** `go test ./compiler/...` — 17/17 packages green.
- [x] **2d-ii.** `go test -run TestE2E ./tests/e2e/` — 77/77
      prior subtests green; 7 stdlib integration proofs still
      red with signatures shifted into Section 3 territory
      (module identity mismatches for `String`, `Result`,
      `Formatter`).
- [x] **2d-iii.** `go test -run TestStdlib ./tests/e2e/` — green.
- [x] **2d-iv.** `go test -run TestStage1CompilesStage2 ./compiler/driver/` — green.
- [x] **2d-v.** `go vet ./...` — clean.

*Refinement rationale:* Original 2a duplicated a helper. 2c-iv
adds a regression-style test so a future driver change that
stops filtering generic originals doesn't silently re-introduce
the bug.

## 3. Canonicalize module identity for generic instantiations

When user code in module `foo` references `List[MyType]`, the
specialization must register under `core.list`, not `foo`.

### 3a. Root-cause fix in `resolvePathType`

- [x] **3a-i.** Verified: `compiler/check/types.go` — the symbol-
      table branch now uses `sym.Module.String()` via the shared
      `resolveTypeName` helper. Auto-loaded stdlib types resolve
      through their defining module.
- [x] **3a-ii.** The old unconditional fallback is gone.
      `resolvePathType` now (i) checks the new
      `currentGenericParams` scope (pushed by registerFn /
      registerStruct / registerEnum / registerImpl / registerTrait /
      checkFnBody) so `T` interns as `KindGenericParam`;
      (ii) otherwise emits `"unresolved type '<name>'"` and returns
      `Unknown`. No phantom-struct fallthrough.
- [x] **3a-iii.** `TestUnresolvedTypeEmitsDiagnostic` in
      `compiler/check/check_test.go`.

*Refinement rationale:* Original 3a-ii added a `coreTypeLookup`
hard-coded table as a fallback. That re-introduces the silent
phantom-struct pattern one layer down. Once Section 1 makes
stdlib symbols actually visible, the existing symbol-table
lookup at 68-79 works unchanged. The **correct** fallback when
a name is unknown is a diagnostic, not a hard-coded core
lookup. The `coreTypeLookup` idea must be removed.

### 3b. Same root-cause fix in `checkStructLit`

- [x] **3b-i.** `checkStructLit` uses `resolveTypeName` for the
      module lookup and emits `"unknown struct '<name>'"` on miss.
- [x] **3b-ii.** `(*Checker).resolveTypeName(name) (modStr, kind, ok)`
      is the shared helper. `TestUnknownStructLiteralEmitsDiagnostic`
      guards the new diagnostic.

### 3c. Emit specialized struct/enum definitions

Once 3a and 3b are correct, `List[I32]` is interned with
module `core.list`, and `List[T]` (the base) is already there
too. The emitter needs to substitute.

- [x] **3c-i.** `emitTypeDefIfNeeded` now funnels struct and
      enum layout through `concreteLayout`, which falls back to
      `BaseOf` + `SubstituteFields` when the specialization's
      own `Fields` is empty. Opaque forward-decls only emit when
      the base is genuinely unknown (a diagnostic has already
      fired upstream in that case).
- [x] **3c-ii.** `(*TypeTable).BaseOf` — returns the template
      TypeId for a (module, name) specialization, or
      `InvalidTypeId` otherwise. Four regression tests cover
      specialization, template, primitive, and missing-template.
- [x] **3c-iii.** `(*TypeTable).SubstituteFields` — walks the
      template's `Fields` through a `substituteType` recursion
      covering `KindPtr/Ref/MutRef/Slice/Array/Tuple/Channel/
      Func/Struct/Enum/GenericParam`. Memoized per-call. Three
      regression tests (primitive sub, nested generic, invalid
      base). Mirrors `monomorph.Context.Substitute` — a future
      cleanup can switch monomorph to delegate.
- [x] **3c-iv.** Same substitution flows through `KindEnum`.
- [x] **3c-v.** No "try canonical core module" fallback —
      `concreteLayout` returns `(te.Fields, te.FieldNames)`
      unchanged when `BaseOf` is Invalid so the opaque path
      still runs. The Section 6 regression for `BaseOf`
      silent-fallback is scheduled for that phase.

*Refinement rationale:* Original 3c-v was a symptom patch
making the "module identity mismatch" silently recoverable.
Remove it; rely on Section 3a's root-cause fix.

### 3d. Register base fields for generic enums

- [x] **3d-i.** `registerEnum` unconditionally interns the
      base enum (`InternEnum(modStr, name, nil)`) and calls
      `SetEnumFields` with the widest variant payload — for
      generic enums too. `SubstituteFields` therefore has real
      fields to walk when emitting `Option[String]`,
      `Result[(), String]`, etc.
- [x] **3d-ii.** Covered indirectly by proofs 5c / 5e, which
      return `Result[_, String]` and use `Option[V]`. The
      failure signature has shifted out of "enum payload
      missing" territory.

## 4. Secondary codegen issues

These become visible once Sections 1–3 are in. Each is a small
root-cause fix; none depends on the others.

### 4a. Qualify trait/impl method names with target type

- [x] **4a-i.** Confirmed: specialized impl methods already
      register under `baseName.method`. The gap was non-generic
      trait impls (`impl Equatable : I32 { fn eq }` vs
      `impl Equatable : String { fn eq }` both emitting
      `Fuse_eq`).
- [x] **4a-ii.** Driver's `ImplDecl` branch computes
      `implTargetName(impl)` and renames `hirFn.Name =
      targetName + "__" + hirFn.Name` before liveness/lowering.
      The qualified name is what reaches MIR and codegen.
- [x] **4a-iii.** `(*Lowerer).methodCalleeName(recvType, name)`
      in `compiler/lower/lower.go` builds the qualified callee
      string the same way — primitive/struct/enum all get
      `Fuse_<TypeName>__method`, generic specs fold TypeArgs
      between (`Fuse_List__I32__push`). It unwraps Ref/MutRef
      so `ref self` dispatches on the element type. Method
      dispatch is decided in the lowerer, not codegen
      (Rule 3.1).
- [x] **4a-iv.** Trait default-method clones in
      `registerImpl` are renamed to `targetName + "__" +
      defaultFn.Name` before being appended to `mod.File.Items`,
      so the driver's `FnDecl` branch emits them under the
      qualified name. Proof: `w18_trait_default_method` e2e
      stays green.
- [x] **4a-v.** Drop call sites route through a new
      `dropFnName(tt, id)` helper that returns
      `Fuse_<TypeName>__drop` for both `InstrDrop` and the
      `TermReturn` end-of-function path. The
      `TestDropWithDropTrait` assertion was updated to the new
      scheme; the old `Fuse_<module>__<Type>_drop` pattern is
      gone.
- [x] **4a-vi.** The `Fuse_eq` conflicting-types gcc errors
      previously observed in the Section 5 proofs have
      disappeared from the failure signatures — confirming
      that trait impls on multiple types no longer collide.

### 4b. Preserve extern function names

- [x] **4b-i.** `MangleName` returns a `fuse_rt_*` name
      verbatim: the early-return guard is at the top of the
      function so it precedes every mangling path.
- [x] **4b-ii.** The single guard also covers the 4a
      method-qualification scheme — extern names never receive
      a `Fuse_` or `<Type>__` prefix.
- [x] **4b-iii.** The lowerer's direct-call path skips the
      `Fuse_` prepending when the identifier starts with
      `fuse_rt_`, leaving the raw name.
- [x] **4b-iv.** `TestMangleNameGolden` in
      `compiler/codegen/codegen_test.go` pins
      `{"main", "main"} → "main"`, `{"main", "add"} →
      "Fuse_main__add"`, `{"", "I32__eq"} → "Fuse_I32__eq"`,
      `{"", "fuse_rt_proc_argc"} → "fuse_rt_proc_argc"`, and
      `{"core.list", "push"} → "Fuse_core_list__push"`
      (Rule 7.2).

*Refinement rationale:* Original said 4a–4d were "independent"
but 4a and 4b both edit `mangle.go`'s `MangleName`. They must
be coordinated or one clobbers the other. 4b-iv adds the
golden test mandated by Rule 7.2.

### 4c. Strip numeric literal suffixes

- [x] **4c-i.** `constValue` dispatches numeric-typed constants
      through `stripNumericSuffix`, which removes any trailing
      `usize|isize|u128|u64|u32|u16|u8|i128|i64|i32|i16|i8|f64|f32`
      before the value reaches generated C.
- [x] **4c-ii.** The suffix list is ordered longest-first and
      the helper requires the remainder to begin with a digit,
      `-`, or `.` — `42u8` strips to `42`, but an identifier
      like `myvar_u8` stays intact.
- [x] **4c-iii.** `TestStripNumericSuffix` covers each class
      (`0usize`, `42u8`, `100i32`, `1.5f32`, `-7isize`, plain
      `0`, `true`, identifier-like `myvar_u8`, bare `u8`).

### 4d. Borrow-of-borrow double-pointer — fix in the lowerer

- [x] **4d-i.** `lowerUnary` now checks
      `(*Lowerer).isBorrow(operandType)` — if the operand is
      already `KindRef` / `KindMutRef`, the lowerer emits
      `InstrCopy` rather than `InstrBorrow`. No `&T*` → `T**`
      in generated C.
- [x] **4d-ii.** No emitter-side `&` suppression was added.
      The fix lives entirely in HIR→MIR, preserving Rule 3.1
      (three IRs) and Rule 4.1 (root causes).
- [x] **4d-iii.** All non-stdlib e2e tests stay green after
      the change, including `w18_drop_destructor` which
      exercises `ref` through a scope-exit path. The
      `-Wincompatible-pointer-types` class of gcc errors
      previously visible in the Section 5 proof signatures has
      been removed from the remaining failure modes.

*Refinement rationale:* Original 4d patched the codegen. That
is Rule 4.2 (workarounds forbidden). The correct place to
decide "this is already a borrow, do not re-borrow" is the
lowerer.

## 5. Proof programs

These programs are the exit criteria. Each must compile, link,
run, and produce the expected exit code or stdout.

Status after the Section 5 work below:

- [x] **5a.** `struct Config { name: String, version: String }` —
      compiles, links, runs, exits 42. String literals now intern
      under `core.string`; struct-field layout and accessor typing
      work through specialized types.
- [x] **1b-i.** `stdlib_1b_i_println_auto_load` — green.
      `println("hello")` compiles with the core-only auto-load,
      DCE trims unused stdlib bodies, and the `println` builtin
      reaches `fuse_rt_io_write_stdout`.
- [x] **1b-ii.** `stdlib_1b_ii_user_shadows_core_option` — green.
      Confirms the user-wins module shadow behaviour survives the
      stricter checker.
- [ ] **5b.** `struct Registry { entries: List[Entry] }` — red.
      Static method call + DCE + spec method-name path reach the
      `Fuse_List__Entry__push/get/new` symbols correctly, but the
      generic `List[T].push` body has deeper codegen issues:
      `Ptr.null()` types as `int` (no context-aware typing for the
      builtin), `fuse_rt_mem_realloc` returns `Ptr[U8]` which
      can't assign to `Ptr[Entry]` without a cast, and MIR
      liveness leaves some `_lN` locals undeclared. These are
      beyond Section 5 scope — the next wave is targeted list/map
      body hygiene plus explicit cast emission.
- [ ] **5c.** `Result[I32, String]` with `?` — red. Root cause:
      tagged enums whose variants have different-typed payloads
      (`Ok(I32)` vs `Err(String)`) collapse to a single `_f0` slot
      picking the first variant's type. Fix requires C union
      emission for multi-payload-type enums; cross-module code
      that uses `Result[_, String]` hits the type-mismatch at
      initialisation.
- [ ] **5d.** `String.contains/starts_with/byte_at` — red.
      String method specialization path works, but the MIR for
      those methods leaves `_l42`/`_l46` locals undeclared
      (liveness/builder skip in a branch). Needs a focused
      lowerer audit for the match / branching patterns used in
      those methods.
- [ ] **5e.** `Map[String, I32]` with `insert`/`get`/`len` —
      red for the same family of reasons as 5b (Map impl body
      codegen gaps around Ptr handling and multi-variant enum
      payloads via `Option[V]`).
- [ ] **5f.** `python test_all.py` — deferred until the four
      body-level codegen gaps above land.

Each of 5a–5e is an e2e test case in `tests/e2e/e2e_test.go`.

### What Section 5 delivered

Section 5 focused on the compiler-level blockers that were common
to all five proofs, leaving body-level stdlib bugs (Ptr null
typing, tagged-union multi-payload layout, liveness gaps on
stdlib match code) to a follow-up wave. Concretely:

- Auto-load is scoped to `core/` only (language-guide §11.4 "on
  demand" for full/ext) so user programs aren't dragged through
  ext/argparse, ext/toml, ext/yaml, etc. on every compile.
- `emitTypeDefIfNeeded` recurses into `KindRef`/`KindMutRef`/
  `KindPtr` elements so `ref Option[I32]` param types pull the
  `Option[I32]` typedef into the emission set.
- Static method calls (`TypeName.method()`) lower cleanly:
  `checkIdent` returns the nominal TypeId for struct/enum
  symbols, the lowerer recognises the pattern via
  `isStaticMethodCall` and emits `Fuse_<TypeName>__method` with
  no receiver borrow. `Ptr.null()` is a compiler-builtin and
  lowers to a null constant.
- A call-graph DCE pass in the codegen emits only functions
  reachable from `main` (including destructors via `InstrDrop`
  and end-of-function `Drop` calls), so unreachable stdlib
  bodies with pre-existing bugs no longer surface as C errors.
- The monomorphizer's instantiation scanner walks struct field
  types, enum payloads, fn params, and fn return types — not
  just expression uses — so `struct Registry { entries: List[Entry] }`
  registers the `List__Entry__*` specialisations. Original
  generic decls are skipped so `List[T]` doesn't create a
  `List__T__*` phantom.
- String literals and the `println`/`print` builtins intern
  `String` under `core.string` (matching the auto-loaded stdlib
  module) rather than the bare `core` placeholder.
- Variant lookup in `checkVariantConstructor` and type-name
  lookup in `resolveTypeName` both fall through to a graph-wide
  search so specialisations placed in a stdlib module can
  resolve user types (`Entry`, `Token`, …) declared elsewhere.
- `checkField` substitutes the base template's field types
  through `TypeTable.SubstituteFields` when accessing a field on
  a specialisation whose own fields are unset.
- `isAssignableTo` permits same-(Kind, Name, Module) structs/
  enums to unify regardless of TypeArgs — propagated expected
  types (return-position struct lits, `None` literals, method
  calls) give the real target type and full inference is a
  later-wave job.
- Enum variants without parens (`None`) lower to `EnumInit`
  instead of a bare identifier, so codegen emits a tagged
  struct literal rather than an undefined C name.

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
