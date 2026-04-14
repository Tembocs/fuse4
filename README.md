# Fuse

> A compiled systems language pursuing memory safety, concurrency safety, and developer experience as a first-class constraint.

Fuse is a systems language built around explicit ownership, deterministic destruction, visible mutation, explicit error propagation, and structured concurrency. The active bootstrap compiler in this repository is Stage 1: a Go compiler that lowers Fuse to C11 and then relies on the system C toolchain to produce native binaries. The long-term target remains Stage 2: Fuse compiling itself, after which the bootstrap path is retired.

## What This Repository Is

This repository is `fuse3`, the third production attempt of Fuse. It contains:

- the Stage 1 Go compiler under [`compiler/`](docs/repository-layout.md)
- the bootstrap C runtime under [`runtime/`](docs/repository-layout.md)
- the Fuse standard library under [`stdlib/`](docs/repository-layout.md)
- the self-hosted Stage 2 compiler source under [`stage2/`](docs/repository-layout.md)
- the full documentation and learning record for this attempt under [`docs/`](docs)

This repository is intended to become the archival reference for attempt 3.

## Language Direction

Fuse is built around three non-negotiable goals.

1. Memory safety without a garbage collector.
2. Concurrency safety without a borrow checker.
3. Developer experience where important effects remain visible at the call site.

That means no hidden tracing collector, no hidden async runtime, no invisible mutation, and no silent unsafe escape hatches.

## Status

Fuse is pre-1.0. The language design for this attempt is substantially specified, but the compiler remains under construction. The main architectural and implementation record for `fuse3` lives in the top-level docs set:

- [docs/language-guide.md](docs/language-guide.md)
- [docs/implementation-plan.md](docs/implementation-plan.md)
- [docs/repository-layout.md](docs/repository-layout.md)
- [docs/rules.md](docs/rules.md)
- [docs/learning-log.md](docs/learning-log.md)

## Fuse4 Transfer Set

This repository also contains the rewritten foundational documents for the next production attempt, `fuse4`. Those files live under:

- [docs/meta/fuse4/README.md](docs/meta/fuse4/README.md)
- [docs/meta/fuse4/docs/language-guide.md](docs/meta/fuse4/docs/language-guide.md)
- [docs/meta/fuse4/docs/implementation-plan.md](docs/meta/fuse4/docs/implementation-plan.md)
- [docs/meta/fuse4/docs/repository-layout.md](docs/meta/fuse4/docs/repository-layout.md)
- [docs/meta/fuse4/docs/rules.md](docs/meta/fuse4/docs/rules.md)
- [docs/meta/fuse4/docs/learning-log.md](docs/meta/fuse4/docs/learning-log.md)

Those files are the transfer-ready seed documents for the next repository. The current top-level docs remain the historical record of `fuse3`.

## If You Are Reading This Repo Cold

Read in this order:

1. [docs/rules.md](docs/rules.md)
2. [docs/implementation-plan.md](docs/implementation-plan.md)
3. [docs/language-guide.md](docs/language-guide.md)
4. [docs/repository-layout.md](docs/repository-layout.md)
5. [docs/learning-log.md](docs/learning-log.md)

If your goal is the next repository rather than archival study of this one, start with the `fuse4` transfer set under [docs/meta/fuse4](docs/meta/fuse4).

## License

See [LICENSE](LICENSE).
