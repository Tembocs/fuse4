# Fuse

> A compiled systems programming language focused on memory safety, concurrency safety, and developer experience as a first-class constraint.

Fuse is a statically typed systems language designed for programs that need
predictable behavior, explicit control, and readable semantics. It aims to
provide memory safety without a garbage collector, concurrency safety without a
borrow checker, and APIs whose important effects remain visible at the call
site.

Fuse uses ownership, deterministic destruction, explicit borrowing, and
structured control over mutation and error propagation. There is no hidden
runtime model for ordinary code, no tracing collector, and no requirement to
understand invisible effects before reading a function call.

## What Fuse Emphasizes

- memory safety through ownership analysis and deterministic destruction
- concurrency through channels, ranked synchronization, and explicit thread
  creation
- explicit mutation through `mutref` and related ownership forms
- explicit error propagation through `Result`, `Option`, and `?`
- explicit unsafe boundaries through `unsafe {}` and raw pointer types

The language is intended to make the cost and effect of an operation easier to
see from the code itself. Mutation, fallibility, and unsafe behavior should not
be hidden behind ordinary-looking calls.

## Language Shape

Fuse is compiled ahead of time to native code. The language includes:

- structs, enums, traits, and impl blocks
- generic functions and generic types
- deterministic ownership and borrowing
- pattern matching and expression-oriented control flow
- channels and shared synchronization primitives for concurrency
- a small explicit unsafe boundary for FFI and raw pointer work

The current compiler architecture uses a bootstrap path in which a Go compiler
lowers Fuse through C11 before producing native binaries. The long-term goal is
a self-hosted Fuse compiler that no longer depends on that bootstrap backend.

## Project Documents

The project is organized around five foundational documents:

- `language-guide.md` — the language specification
- `implementation-plan.md` — the build plan from bootstrap to self-hosting
- `repository-layout.md` — the repository structure and placement rules
- `rules.md` — contributor and agent discipline rules
- `learning-log.md` — accumulated lessons from bugs, design gaps, and fixes

Those documents define both the language and the implementation discipline. If
the implementation and the documents disagree, the documents are the place to
start.

## Status

Fuse is pre-1.0 and should be treated as an active language and compiler effort,
not a stable production platform. The language direction is deliberate, but the
implementation is still evolving and the compiler remains under active
construction.

If you are approaching the project for the first time, read `rules.md` first,
then `implementation-plan.md`, then `language-guide.md`.