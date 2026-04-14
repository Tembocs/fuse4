# Fuse Repository Layout

> Status: normative for the next production attempt of Fuse (`fuse4`).
>
> This document defines the physical structure of the future repository. If the
> repository layout diverges from this document, the layout is wrong or the
> document is stale. In either case, the discrepancy must be resolved
> deliberately rather than by drift.

## Table of contents

1. Top-level tree
2. Root-level files
3. `cmd/`
4. `compiler/`
5. `runtime/`
6. `stdlib/`
7. `stage2/`
8. `tests/`
9. `examples/`
10. `tools/`
11. `docs/`
12. CI and workflows
13. Unsafe bridge file list
14. Adding a new top-level directory

## 1. Top-level tree

The future repository is organized as follows.

```text
fuse4/
├── cmd/
│   └── fuse/
├── compiler/
│   ├── diagnostics/
│   ├── typetable/
│   ├── ast/
│   ├── lex/
│   ├── parse/
│   ├── resolve/
│   ├── hir/
│   ├── check/
│   ├── liveness/
│   ├── lower/
│   ├── mir/
│   ├── monomorph/
│   ├── codegen/
│   ├── cc/
│   ├── driver/
│   ├── fmt/
│   ├── doc/
│   ├── repl/
│   └── testrunner/
├── runtime/
│   ├── include/
│   ├── src/
│   ├── platform/
│   └── tests/
├── stdlib/
│   ├── core/
│   ├── full/
│   └── ext/
├── stage2/
│   ├── src/
│   ├── tests/
│   └── build/
├── tests/
│   ├── fixtures/
│   ├── e2e/
│   ├── bootstrap/
│   └── property/
├── examples/
├── tools/
├── docs/
├── .ci/
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── .gitignore
```

This layout makes the bootstrap boundary explicit.

- `compiler/` contains the Stage 1 Go compiler.
- `runtime/` contains the bootstrap runtime implemented in C.
- `stage2/` contains the Fuse compiler implemented in Fuse.

When Stage 2 becomes self-hosting and Go and C are retired from the compiler
implementation path, this layout may evolve, but only through an explicit change
to this document and the implementation plan.

## 2. Root-level files

### `go.mod`

Declares the Go module used by the Stage 1 compiler. During bootstrap the Go
module is required and remains source-controlled.

### `go.sum`

Present for Go tooling compatibility. The project policy is zero external Go
dependencies, so this file should remain empty or near-empty unless the policy
changes explicitly.

### `Makefile`

The top-level entry point for local builds. At minimum it must expose:

- `all`
- `stage1`
- `runtime`
- `test`
- `clean`
- `fmt`
- `docs`
- `repro`

### `README.md`

Short project entry point for new readers. It stays brief and points to the core
documents under `docs/`.

### `.gitignore`

Excludes generated artifacts, build output, and editor-local noise. A tracked
source file must never depend on `.gitignore` to disappear from review.

There must be no additional root-level documentation files that duplicate the
role of the core docs. The project uses `docs/` as the normative documentation
surface.

## 3. `cmd/`

`cmd/` contains Go `main` packages during bootstrap.

### `cmd/fuse/`

This is the Stage 1 CLI entry point. It is intentionally thin and owns:

- command-line parsing
- subcommand dispatch
- process exit codes
- stdout and stderr policy
- signal-handling or process-level wiring if required

Business logic does not live in `cmd/fuse/`. Compiler functionality belongs in
`compiler/`.

If a second command-line tool is added later, it must appear as its own
subdirectory under `cmd/`.

## 4. `compiler/`

`compiler/` contains the Stage 1 Go compiler packages. Each subdirectory is a Go
package with one narrow responsibility.

The intended dependency direction is:

```text
diagnostics < typetable < ast < lex < parse < resolve < hir < check < liveness
                         < lower < mir < codegen < cc < driver
                                                                                                        < fmt
                                                                                                        < doc
                                                                                                        < repl
                                                                                                        < testrunner
monomorph spans the semantic and lowering boundary.
```

### `compiler/diagnostics/`

Owns spans, severities, diagnostics, and rendering. Every other package reports
human-facing errors through this package.

### `compiler/typetable/`

Owns interned type identity via `TypeId`. Equality of types is integer
comparison over interned entries, not structural re-walks.

### `compiler/ast/`

Owns the syntax-only AST. AST nodes do not contain resolved symbols, inferred
types, liveness, or backend metadata.

### `compiler/lex/`

Owns lexical analysis.

### `compiler/parse/`

Owns parsing from tokens to AST.

### `compiler/resolve/`

Owns module discovery, symbol indexing, scopes, import resolution, and HIR
construction inputs.

### `compiler/hir/`

Owns the semantically rich HIR and HIR builders. HIR nodes carry metadata fields
used by later passes.

### `compiler/check/`

Owns semantic analysis and type checking. This pass must type-check both user and
stdlib bodies.

### `compiler/liveness/`

Owns the single liveness computation.

### `compiler/lower/`

Owns HIR-to-MIR lowering and the preservation of semantic contracts into MIR.

### `compiler/mir/`

Owns the explicit mid-level IR used by backends.

### `compiler/monomorph/`

Owns generic specialization collection and specialization identity.

### `compiler/codegen/`

Owns the bootstrap backend. During bootstrap this is the C11 emitter.

### `compiler/cc/`

Owns C compiler detection, invocation, and backend-toolchain interaction.

### `compiler/driver/`

Owns the end-to-end orchestration of the Stage 1 compiler pipeline.

### `compiler/fmt/`

Owns formatting support.

### `compiler/doc/`

Owns documentation extraction and rendering.

### `compiler/repl/`

Owns the REPL surface if the project retains one.

### `compiler/testrunner/`

Owns compiler-driven test execution workflows beyond ordinary Go tests.

No additional compiler package may be created casually. A new package must have
one clear responsibility and a defined dependency place in the overall graph.

## 5. `runtime/`

`runtime/` contains the bootstrap runtime implemented in C.

### `runtime/include/`

Contains the runtime header that defines the exported bootstrap ABI.

### `runtime/src/`

Contains the implementation of the runtime entry points. The runtime must remain
small, explicit, and reviewable. High-level behavior belongs in Fuse stdlib,
not in the C runtime.

### `runtime/platform/`

Contains platform-specific shims when unavoidable.

### `runtime/tests/`

Contains runtime-specific tests.

The runtime is bootstrap infrastructure. It exists to expose a minimal and
stable surface, not to become a shadow standard library.

## 6. `stdlib/`

`stdlib/` contains the Fuse standard library and is divided into three tiers.

### `stdlib/core/`

The OS-free core tier. It must not depend on hosted functionality.

Expected content includes:

- primitive types and aliases
- core traits
- strings
- collections
- iterators
- formatting
- hash support
- runtime bridge files

### `stdlib/full/`

The hosted standard library tier. It may depend on `core/` and exposes:

- IO
- filesystem
- process and environment
- time
- threading
- synchronization
- channels

### `stdlib/ext/`

Optional extended libraries. This tier may depend on `core/` or `full/` but not
the reverse.

The dependency rule is strict:

`ext -> full -> core`

Reverse dependencies are forbidden.

## 7. `stage2/`

`stage2/` contains the self-hosted Fuse compiler source.

### `stage2/src/`

Contains the Fuse implementation of the compiler. The package structure should
closely mirror the Stage 1 architecture where practical.

### `stage2/tests/`

Contains stage2-specific validation inputs.

### `stage2/build/`

Contains generated build artifacts and is not part of the normative source tree.

`stage2/` is not a toy example directory. It is the primary long-term target of
the project.

## 8. `tests/`

The top-level `tests/` tree holds shared and end-to-end test assets.

### `tests/fixtures/`

Shared corpora and reusable test packages.

### `tests/e2e/`

End-to-end build and execution tests.

### `tests/bootstrap/`

Self-hosting and reproducibility tests.

### `tests/property/`

Property-based tests, especially around IR transformations and semantic
preservation.

Pass-local tests still belong next to the packages they exercise. The top-level
`tests/` tree is for cross-package and end-to-end assets, not as a dumping
ground.

## 9. `examples/`

Contains example Fuse programs intended to compile with ordinary project
workflows. Examples demonstrate language and library usage and serve as light
integration checks.

Examples should remain small and buildable.

## 10. `tools/`

Contains project tooling that supports the repo but is not part of the compiled
compiler pipeline.

Expected tools include:

- `checklog/`
- `checkdoc/`
- `goldens/`
- `repro/`
- `bench/`

Tooling belongs here when it supports repository workflow, verification, or
documentation, not when it is compiler logic in disguise.

## 11. `docs/`

`docs/` contains the normative documents for the project.

Required files:

- `language-guide.md`
- `implementation-plan.md`
- `repository-layout.md`
- `rules.md`
- `learning-log.md`

Optional supporting directories:

- `adr/` for architecture decisions that cannot live in the language guide or
    implementation plan

The project does not use free-form status documents as normative inputs. The
five foundational files above remain the source of truth.

## 12. CI and workflows

CI configuration lives under `.ci/`.

Expected content:

- workflow definitions
- helper scripts
- reproducibility checks

CI must exercise the same invariants the local workflow expects:

- build
- test
- docs validation
- reproducibility or determinism gates when available

## 13. Unsafe bridge file list

Unsafe Fuse files that bridge to the runtime must be enumerated explicitly by
name.

At minimum, the core runtime bridge surface is expected to include files such as:

- `stdlib/core/rt_bridge/alloc.fuse`
- `stdlib/core/rt_bridge/panic.fuse`
- `stdlib/core/rt_bridge/intrinsics.fuse`

If additional unsafe bridge files are required, this section and the rules
document must both be updated.

## 14. Adding a new top-level directory

New top-level directories are not created casually.

Required procedure:

1. Explain why no existing top-level directory is appropriate.
2. Update this document with the new directory and its intended purpose.
3. Update the implementation plan if the new directory changes the build or
     execution model.
4. Get explicit review before landing the change.

The default assumption is that the existing top-level tree is sufficient.