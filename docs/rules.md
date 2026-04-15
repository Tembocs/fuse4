# Fuse Project Rules

> Status: normative for the next production attempt of Fuse (`fuse4`).
>
> This document is the canonical rule set for contributors and AI agents. It is
> intentionally dense. Every rule exists because violating it either damaged the
> architecture or made debugging materially harder in the previous attempt.

## Table of contents

1. Quickstart for contributors and AI agents
2. Language guide precedence
3. Compiler architecture invariants
4. Bug policy
5. Stdlib policy
6. Testing rules
7. Determinism rules
8. External dependency rules
9. Commit and PR rules
10. Learning log rules
11. Multi-machine workflow
12. Safety and `unsafe`
13. Permanent prohibitions
14. AI agent behavior

## 1. Quickstart for contributors and AI agents

Before writing code in this repository, a contributor or AI agent MUST:

1. Read `docs/language-guide.md`.
2. Read `docs/implementation-plan.md` and locate the current wave and phase.
3. Read `docs/repository-layout.md`.
4. Read `docs/learning-log.md`, especially recent entries.
5. Check the working tree state.

Before committing, a contributor or AI agent MUST:

1. Run `fuse check` on touched Fuse code when applicable.
2. Run focused tests for affected packages.
3. Run broader test suites when the change crosses package boundaries.
4. Verify that the diff contains only intended changes.
5. Ensure any non-trivial bug fix has both a regression and a learning-log entry.

## 2. Language guide precedence

### Rule 2.1 — The guide precedes implementation.

No language feature may be implemented before it appears in the language guide.

### Rule 2.2 — The guide is normative.

If the compiler and the guide disagree, the compiler is wrong unless the guide is
explicitly updated first.

### Rule 2.3 — Silence means absence.

If the guide does not specify a feature or behavior, that feature does not exist.

### Rule 2.4 — Implementation contracts are mandatory.

Any language feature that is architecturally sensitive must include an
implementation contract in the guide. Backend-critical semantics must not be left
to implication.

## 3. Compiler architecture invariants

### Rule 3.1 — There are exactly three IRs.

The compiler architecture is `AST -> HIR -> MIR`. No pass may skip across an IR
boundary.

### Rule 3.2 — IR types are disjoint.

`ast.*`, `hir.*`, and `mir.*` are separate type families. They must not be
interchangeable.

### Rule 3.3 — HIR metadata is not optional.

HIR nodes must be built through builders or constructors that establish all
required metadata fields.

### Rule 3.4 — Every pass declares reads and writes.

Passes are registered in a manifest that declares which metadata they consume and
which metadata they produce.

### Rule 3.5 — Invariant walkers run at pass boundaries.

Invariant walkers must run in debug and CI contexts. A failing invariant is a bug
in the producing pass, not a reason to disable the walker.

### Rule 3.6 — Deterministic collections only in IR.

Builtin map iteration must not determine user-visible IR ordering or backend
emission order.

### Rule 3.7 — The TypeTable is global and canonical.

Types are interned and referenced by `TypeId`. Alternate type-identity systems
are forbidden.

### Rule 3.8 — Liveness is computed once.

Liveness is computed once per function and then consumed by later passes. It must
not be recomputed opportunistically downstream.

### Rule 3.9 — Unresolved types must not cross into codegen.

If a type is unresolved after checking and lowering, the compiler must emit a
diagnostic and stop. Silent fallback to `Unknown` or `int` is forbidden.

### Rule 3.10 — Backend contracts are architecture, not cleanup.

Pointer-category separation, total unit erasure, monomorphization completeness,
and structural divergence are design constraints, not codegen conveniences.

## 4. Bug policy

### Rule 4.1 — Fix root causes.

A bug fix must remove the structural cause, not merely quiet the visible symptom.

### Rule 4.2 — No workarounds.

Code that exists only because some other code is wrong is forbidden.

### Rule 4.3 — Every meaningful bug gets a regression.

If the bug took real diagnosis effort or revealed a real invariant, the fix must
land with a regression test.

### Rule 4.4 — Every meaningful bug gets a learning-log entry.

If the bug taught the team something about the design, it belongs in the log.

### Rule 4.5 — Bootstrap tests use the real Stage 2 source.

Self-hosting verification must compile the real `stage2/` compiler, not only toy
inputs.

## 5. Stdlib policy

### Rule 5.1 — Stdlib is the compiler's semantic stress test.

If stdlib fails, assume the compiler is wrong before assuming stdlib is wrong.

### Rule 5.2 — Stdlib bodies must be checked.

Skipping stdlib body checking is forbidden.

### Rule 5.3 — Core is OS-free.

`stdlib/core/` must not depend on hosted APIs.

### Rule 5.4 — Full depends on core; ext depends on full or core.

Dependency direction is one-way.

### Rule 5.5 — No hidden special cases in public stdlib APIs.

If a behavior belongs to a trait or language rule, it must not be quietly hardcoded
in stdlib public API behavior.

### Rule 5.6 — Public stdlib API requires documentation.

Missing docs on public stdlib APIs are a correctness issue for project quality.

## 6. Testing rules

### Rule 6.1 — Tests are deterministic.

Tests must not depend on ambient randomness, wall clock, or machine-local state.

### Rule 6.2 — Golden tests are explicit.

Golden updates require an intentional workflow. Goldens must be byte-stable.

### Rule 6.3 — Test names state the invariant.

A test name should explain what property it protects.

### Rule 6.4 — Property tests must be reproducible.

Property tests require stable seeds and reproducible failures.

### Rule 6.5 — Integration tests should hit the real backend when possible.

A test that claims the compiler emits working binaries should invoke the backend
toolchain and execute the result.

### Rule 6.6 — Stdlib is part of validation, not a separate concern.

Wave exit criteria must include stdlib validation once the relevant surface is in
scope.

### Rule 6.7 — Bootstrap health is release-blocking.

Any regression in self-hosting or reproducibility is a release blocker.

### Rule 6.8 — No feature is complete without an end-to-end proof program.

A feature is complete only when a Fuse program that uses it compiles, links,
runs, and produces the correct output. Unit tests prove a component works in
isolation. End-to-end proof programs prove the compiler works for the user. Both
are required. (See learning-log L013, L014.)

### Rule 6.9 — Stubs must emit diagnostics, not silent defaults.

If a feature is parsed and type-checked but not lowered or codegenned, the
compiler must emit a diagnostic such as `"closures are not yet implemented"`.
A stub that compiles silently is indistinguishable from a working implementation
to both the test suite and the user. Silent stubs are forbidden. (See
learning-log L013.)

### Rule 6.10 — Exit criteria must be behavioral, not only structural.

Wave exit criteria must include at least one behavioral requirement: "this
program compiles, runs, and returns exit code N." Structural criteria ("HIR
nodes carry metadata", "MIR terminates correctly") are necessary but never
sufficient on their own. (See learning-log L014.)

## 7. Determinism rules

### Rule 7.1 — Same input, same bytes.

Identical source, compiler version, and target must produce byte-equivalent
outputs according to project reproducibility rules.

### Rule 7.2 — Symbol mangling is stable.

Mangled names depend only on semantic identity, not iteration order or timestamps.

### Rule 7.3 — No ambient randomness in output-affecting paths.

If unique names are needed, deterministic counters are allowed; random numbers are
not.

### Rule 7.4 — No timestamps or absolute paths in goldens.

Reviewable artifacts must be stable across machines and days.

## 8. External dependency rules

### Rule 8.1 — Zero external Go dependencies.

The Stage 1 compiler must not depend on external Go modules.

### Rule 8.2 — Runtime dependencies are explicit.

Bootstrap dependencies on Go and C are explicit and temporary. Hidden toolchain
dependencies are forbidden.

### Rule 8.3 — No host-language leakage into Fuse artifacts.

Stage 1 implementation details must not become part of compiled Fuse program
semantics.

## 9. Commit and PR rules

### Rule 9.1 — One logical change per commit.

Avoid mixed commits that combine unrelated feature, refactor, and formatting work.

### Rule 9.2 — Commit messages use stable areas.

Commit subjects use the form `<area>: <subject>`.

Valid areas include:

- `compiler/<package>`
- `runtime`
- `stdlib/core`
- `stdlib/full`
- `stdlib/ext`
- `cli`
- `tests`
- `docs`
- `ci`
- `tools`

### Rule 9.3 — Branches reference plan IDs when possible.

Task-scoped work should name the wave or task ID in the branch name.

### Rule 9.4 — Force-pushing shared protected branches is forbidden.

## 10. Learning log rules

### Rule 10.1 — The learning log is append-only.

Do not rewrite old entries. Supersede them with new ones.

### Rule 10.2 — The format is enforced.

Each entry must include reproducer, root cause, spec gap, plan gap, fix,
cascading effects, architectural lesson, and verification.

### Rule 10.3 — The learning log feeds the guide and plan.

If a bug exposed a specification or planning hole, the corresponding document
must be updated.

## 11. Multi-machine workflow

### Rule 11.1 — Commit before changing machines.

Uncommitted work across machines causes silent drift and duplicated debugging.

### Rule 11.2 — Re-validate after context switches.

When resuming work after a machine or session switch, re-read the relevant docs,
plan section, and recent learning-log entries.

## 12. Safety and `unsafe`

### Rule 12.1 — Unsafe remains explicit.

Unsafe operations must remain visible at the use site.

### Rule 12.2 — Unsafe bridge files are enumerated by name.

New unsafe bridge files require explicit documentation updates.

### Rule 12.3 — Public safe APIs must not smuggle unsafe behavior.

If a safe wrapper exists, its safety story must be defensible and documented.

## 13. Permanent prohibitions

The following are permanently forbidden unless the foundational docs change
explicitly.

- implementing undocumented language features
- bypassing stdlib body checking
- introducing workarounds in stdlib for compiler bugs
- recomputing liveness in backend passes
- letting unresolved types reach codegen
- depending on nondeterministic IR ordering
- introducing external Go dependencies into Stage 1
- treating bootstrap C11 details as permanent language semantics

## 14. AI agent behavior

An AI agent working on this repository MUST:

- read the foundational docs before making architectural changes
- prefer root-cause fixes over local patches
- avoid destructive actions on unrelated user work
- update tests and the learning log when required
- treat the language guide and implementation plan as the source of truth
- preserve the bootstrap model until the plan explicitly retires it