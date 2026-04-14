# Fuse4 Documentation Transfer Set

This directory contains the replacement foundational documents for the next
production attempt of Fuse.

These files are intentionally isolated from the top-level `docs/` tree because
this repository (`fuse3`) will be archived as historical reference. The files
under `docs/meta/fuse4/docs/` are the transfer-ready documents intended to be
copied into the new repository.

## Transfer Rule

When the new repository is created, copy the files from:

`docs/meta/fuse4/docs/`

to:

`docs/`

in the new repository.

The current top-level `docs/` tree in `fuse3` remains the archival record of
attempt 3 and must not be overwritten by these rewrite documents.

## Scope

This subtree contains the five foundational documents required by
`docs/meta/document_writing_brief.md`:

1. `language-guide.md`
2. `implementation-plan.md`
3. `repository-layout.md`
4. `rules.md`
5. `learning-log.md`

## Fixed Bootstrap Model

The rewrite preserves the project's non-negotiable bootstrap architecture:

- Stage 1 compiler: Go
- Runtime: C
- Stage 2 compiler: Fuse
- Terminal goal: Fuse compiles itself, after which Go and C are retired from
  the compiler implementation path

The current C11 backend is a bootstrap strategy, not the terminal architecture.

## Naming Conventions For The New Plan

The new implementation plan uses explicit, globally unique identifiers.

- Wave headings:
  `Wave 04: Type Checking and Semantic Validation`
- Phase headings:
  `Phase 03: Trait Resolution and Bound Dispatch [W04-P03-TRAIT-RESOLUTION]`
- Task headings:
  `Task 01: Register All Function Types Before Body Checking [W04-P03-T01-FN-TYPE-REGISTRATION]`

Numbers are zero-padded to keep references stable and sortable.

## Authoring Order

Documents must be authored in this order:

1. `language-guide.md`
2. `implementation-plan.md`
3. `repository-layout.md`
4. `rules.md`
5. `learning-log.md`

That order is mandatory because the later documents depend on the earlier ones.