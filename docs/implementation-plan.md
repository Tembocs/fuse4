# Fuse Implementation Plan

> Status: normative for the next production attempt of Fuse (`fuse4`).
>
> This document is the build plan from an empty repository to a self-hosting
> Fuse compiler and the later retirement of bootstrap-only implementation
> languages.

## Overview

Fuse is implemented in stages.

- Stage 1 compiler: Go
- Runtime during bootstrap: C
- Stage 2 compiler: Fuse

The bootstrap stack is fixed. Go and C are allowed during bootstrap because the
project must reach a self-hosted Fuse compiler as quickly and safely as possible.
After Stage 2 compiles itself reliably and a native backend is stable, Go and C
are retired from the compiler implementation path.

The C11 backend is therefore bootstrap infrastructure, not the terminal backend.
Design decisions in HIR, MIR, type identity, ownership analysis, and pass
structure must not depend on C11 in a way that would block the later native
backend.

## Working principles

1. Correctness precedes velocity.
2. Structural fixes beat symptom fixes.
3. No workarounds are allowed in compiler, runtime, or stdlib.
4. Stdlib is the compiler's semantic stress test.
5. Every wave has explicit entry and exit criteria.
6. Every task must be small enough to review and verify directly.

## Naming conventions

The plan uses globally unique identifiers.

- Wave headings:
  `Wave 04: Type Checking and Semantic Validation`
- Phase headings:
  `Phase 03: Trait Resolution and Bound Dispatch [W04-P03-TRAIT-RESOLUTION]`
- Task headings:
  `Task 01: Register All Function Types Before Body Checking [W04-P03-T01-FN-TYPE-REGISTRATION]`

All wave, phase, and task numbers are zero-padded.

## Task format

Every task in this plan must be written with:

- a short goal
- an exact definition of done
- required regression coverage
- clear scope boundaries

In this document, task bullets use the compact form:

`Task 01: ... [Wxx-Pyy-Tzz-...]`
`DoD:` verifiable completion rule.

## Waves at a glance

| Wave | Theme | Entry criterion | Exit criterion |
|---|---|---|---|
| 00 | Project foundations | — | build, test, CI, and docs scaffold exist |
| 01 | Lexer | Wave 00 done | every token kind and lexical ambiguity covered |
| 02 | Parser and AST | Wave 01 done | all language constructs parse deterministically |
| 03 | Module graph and resolution | Wave 02 done | module graph, imports, and symbols are resolved |
| 04 | TypeTable, HIR, pass manifest | Wave 03 done | typed HIR shape and pass graph are enforced |
| 05 | Type checker | Wave 04 done | stdlib and user bodies type-check with no unknowns |
| 06 | Ownership and liveness | Wave 05 done | single liveness computation and destruction rules hold |
| 07 | HIR to MIR lowering | Wave 06 done | MIR is structurally correct and property-tested |
| 08 | Runtime library | Wave 00 done | bootstrap runtime surface is implemented |
| 09 | C11 backend and contracts | Waves 07 and 08 done | generated C is structurally correct and deterministic |
| 10 | Native build driver and linking | Wave 09 done | end-to-end `fuse build` links working binaries |
| 11 | CLI, diagnostics, workflows | Wave 10 done | user-facing compiler workflow is coherent |
| 12 | Core stdlib | Wave 11 done | core tier ships and stress-tests frontend and backend |
| 13 | Hosted stdlib | Wave 12 done | OS-facing surface and concurrency tier ship |
| 14 | Stage 2 port | Wave 13 done | stage1 compiles stage2 successfully |
| 15 | Self-hosting gate | Wave 14 done | stage2 compiles itself reproducibly |
| 16 | Native backend transition | Wave 15 done | bootstrap C11 dependency is removed |
| 17 | Retirement of Go and C | Wave 16 done | Fuse owns the compiler implementation path |
| 18 | Targets and ecosystem | Wave 17 done | cross-target and library growth resume on native base |

## Wave 00: Project Foundations

Goal: establish the repository, module, build, test, tooling, and documentation
foundations required for disciplined compiler work.

Entry criterion: none.

Exit criteria:

- `make all` succeeds from a clean checkout
- `go test ./...` succeeds on the initial package set
- CI runs on every push and PR
- the five foundational docs exist and are readable

### Phase 01: Repository Initialization [W00-P01-REPOSITORY-INITIALIZATION]

- Task 01: Create repository skeleton [W00-P01-T01-REPO-SKELETON]
  DoD: top-level source directories and required seed files exist.
- Task 02: Add foundational docs [W00-P01-T02-FOUNDATIONAL-DOCS]
  DoD: language guide, implementation plan, repository layout, rules, and
  learning log exist.
- Task 03: Establish archival and generated-file policy
  [W00-P01-T03-ARTIFACT-POLICY]
  DoD: generated artifacts are excluded and source-of-truth files are explicit.

### Phase 02: Go Module and Build Scaffold [W00-P02-GO-MODULE-AND-BUILD-SCAFFOLD]

- Task 01: Initialize Go module [W00-P02-T01-GO-MOD]
  DoD: `go.mod` exists and `go build ./...` works.
- Task 02: Create package stubs [W00-P02-T02-PACKAGE-STUBS]
  DoD: all planned Stage 1 packages compile as empty packages.
- Task 03: Create Stage 1 CLI stub [W00-P02-T03-CLI-STUB]
  DoD: `go run ./cmd/fuse` prints a controlled not-yet-implemented message.

### Phase 03: Build and CI Baseline [W00-P03-BUILD-AND-CI-BASELINE]

- Task 01: Author Makefile targets [W00-P03-T01-MAKEFILE]
  DoD: `all`, `stage1`, `runtime`, `test`, `clean`, `fmt`, and `docs` work.
- Task 02: Add CI matrix [W00-P03-T02-CI-MATRIX]
  DoD: Linux, macOS, and Windows builds run.
- Task 03: Add golden harness [W00-P03-T03-GOLDEN-HARNESS]
  DoD: at least one checked-in golden test passes.

## Wave 01: Lexing and Tokenization

Goal: build a deterministic lexer that covers the full token set and all known
lexical ambiguities.

Entry criterion: Wave 00 done.

Exit criteria:

- every token kind in the language guide is tested
- BOM is rejected
- nested block comments work
- raw strings obey the full-pattern rule
- `?.` is emitted as one token

### Phase 01: Token Model [W01-P01-TOKEN-MODEL]

- Task 01: Define token kinds [W01-P01-T01-TOKEN-KINDS]
  DoD: token enumeration covers punctuation, operators, literals, and keywords.
- Task 02: Define span model [W01-P01-T02-SPAN-MODEL]
  DoD: each token carries stable source spans.

### Phase 02: Scanner Core [W01-P02-SCANNER-CORE]

- Task 01: Implement identifier and keyword scanning
  [W01-P02-T01-IDENTIFIERS-AND-KEYWORDS]
  DoD: reserved and active keywords tokenize correctly.
- Task 02: Implement literal scanning [W01-P02-T02-LITERALS]
  DoD: integer, float, string, and raw string forms tokenize correctly.
- Task 03: Implement comments and trivia [W01-P02-T03-COMMENTS-AND-TRIVIA]
  DoD: line comments, nested block comments, and whitespace are handled.

### Phase 03: Lexical Edge Cases [W01-P03-LEXICAL-EDGE-CASES]

- Task 01: Enforce raw-string full-pattern recognition
  [W01-P03-T01-RAW-STRING-GUARD]
  DoD: `r#abc` does not enter raw-string mode.
- Task 02: Enforce `?.` longest-match tokenization
  [W01-P03-T02-OPTIONAL-CHAIN-TOKEN]
  DoD: parser receives a single `?.` token.
- Task 03: Add lexer golden and property tests [W01-P03-T03-LEXER-TESTS]
  DoD: deterministic lexing corpus exists and reprints stably.

## Wave 02: Parser and AST Construction

Goal: build an AST-only parser that accepts the full surface grammar without
semantic shortcuts.

Entry criterion: Wave 01 done.

Exit criteria:

- parser handles every grammar construct in the guide
- parser does not panic on malformed input
- AST remains syntax-only

### Phase 01: AST Surface [W02-P01-AST-SURFACE]

- Task 01: Define AST node set [W02-P01-T01-AST-NODE-SET]
  DoD: AST node kinds are exhaustive and syntax-only.
- Task 02: Define AST builders or constructors [W02-P01-T02-AST-CONSTRUCTION]
  DoD: AST creation is consistent and span-correct.

### Phase 02: Core Parsing [W02-P02-CORE-PARSING]

- Task 01: Parse items and declarations [W02-P02-T01-ITEMS-AND-DECLS]
  DoD: functions, structs, enums, traits, impls, consts, and imports parse.
- Task 02: Parse expressions and statements [W02-P02-T02-EXPRS-AND-STMTS]
  DoD: precedence and associativity are correct.
- Task 03: Parse type expressions [W02-P02-T03-TYPE-EXPRS]
  DoD: tuples, arrays, slices, pointers, and generics parse.

### Phase 03: Ambiguity Control [W02-P03-AMBIGUITY-CONTROL]

- Task 01: Implement struct-literal disambiguation
  [W02-P03-T01-STRUCT-LITERAL-DISAMBIGUATION]
  DoD: `IDENT {` is parsed as struct literal only when syntactically valid.
- Task 02: Handle optional chaining parse forms
  [W02-P03-T02-OPTIONAL-CHAIN-PARSE]
  DoD: `expr?.field` parses correctly.
- Task 03: Add parser regression corpus [W02-P03-T03-PARSER-REGRESSIONS]
  DoD: ambiguity regressions are covered by tests and goldens.

## Wave 03: Name Resolution and Module Graph

Goal: resolve symbols, imports, and the module graph without semantic leakage
into later IR layers.

Entry criterion: Wave 02 done.

Exit criteria:

- package discovery is deterministic
- import cycles are diagnosed
- qualified enum variant access resolves
- import resolution supports module-first fallback to module-plus-item

### Phase 01: Package Discovery [W03-P01-PACKAGE-DISCOVERY]

- Task 01: Discover module tree [W03-P01-T01-DISCOVER-MODULES]
  DoD: source files map deterministically to module paths.
- Task 02: Build initial module graph [W03-P01-T02-MODULE-GRAPH]
  DoD: modules and imports are collected without semantic resolution.

### Phase 02: Symbol Infrastructure [W03-P02-SYMBOL-INFRASTRUCTURE]

- Task 01: Create symbol table and scopes [W03-P02-T01-SYMBOL-TABLE]
  DoD: nested scope and module-scope lookup work.
- Task 02: Index top-level symbols [W03-P02-T02-TOP-LEVEL-INDEX]
  DoD: items, variants, and exported names are present.

### Phase 03: Import and Path Resolution [W03-P03-IMPORT-AND-PATH-RESOLUTION]

- Task 01: Resolve imports with module-first fallback
  [W03-P03-T01-MODULE-FIRST-IMPORT-RESOLUTION]
  DoD: `import util.math.Pair` resolves as item import when needed.
- Task 02: Support qualified enum variant access
  [W03-P03-T02-QUALIFIED-ENUM-VARIANTS]
  DoD: `EnumName.Variant` resolves in expressions and patterns.
- Task 03: Detect import cycles [W03-P03-T03-IMPORT-CYCLE-DETECTION]
  DoD: cyclic imports are diagnosed with a readable cycle path.

## Wave 04: TypeTable, HIR, and Pass Manifest

Goal: establish the typed semantic IR surface and the pass graph that later
stages rely on.

Entry criterion: Wave 03 done.

Exit criteria:

- HIR exists and is distinct from AST and MIR
- HIR builders enforce metadata shape
- pass manifest validates declared metadata dependencies
- invariant walkers are in place

### Phase 01: Global TypeTable [W04-P01-GLOBAL-TYPETABLE]

- Task 01: Define TypeId and TypeTable [W04-P01-T01-TYPEID-AND-TABLE]
  DoD: all later IR layers refer to types by interned IDs.
- Task 02: Encode nominal identity [W04-P01-T02-NOMINAL-IDENTITY-ENCODING]
  DoD: defining symbol and concrete args are part of type identity.

### Phase 02: HIR Builders and Metadata [W04-P02-HIR-BUILDERS-AND-METADATA]

- Task 01: Define HIR node set [W04-P02-T01-HIR-NODE-SET]
  DoD: HIR node kinds are exhaustive and semantically oriented.
- Task 02: Add per-node metadata [W04-P02-T02-HIR-METADATA]
  DoD: type, ownership, liveness hooks, divergence, and context fields exist.
- Task 03: Enforce builder-only construction [W04-P02-T03-BUILDER-ENFORCEMENT]
  DoD: HIR cannot be built ad hoc without metadata defaults.

### Phase 03: Pass Graph and Invariants [W04-P03-PASS-GRAPH-AND-INVARIANTS]

- Task 01: Define pass manifest [W04-P03-T01-PASS-MANIFEST]
  DoD: passes declare reads and writes explicitly.
- Task 02: Implement invariant walkers [W04-P03-T02-INVARIANT-WALKERS]
  DoD: post-pass structural checks run in debug and CI modes.
- Task 03: Prohibit nondeterministic IR collections
  [W04-P03-T03-DETERMINISTIC-IR]
  DoD: HIR and MIR do not rely on builtin map iteration order.

## Wave 05: Type Checking and Semantic Validation

Goal: build a checker that fully types user code and stdlib bodies without
leaving unknown metadata for later passes.

Entry criterion: Wave 04 done.

Exit criteria:

- no checked HIR node retains `Unknown` type metadata
- all function declaration nodes are typed before body checking
- stdlib bodies are checked in the same pass as user modules
- trait-bound lookup works through supertraits
- contextual generic inference works in constructor-style calls

### Phase 01: Function Type Registration [W05-P01-FN-TYPE-REGISTRATION]

- Task 01: Index all function signatures [W05-P01-T01-INDEX-FN-SIGNATURES]
  DoD: top-level functions, impl methods, and externs all receive function types.
- Task 02: Separate signature registration from body checking
  [W05-P01-T02-TWO-PASS-CHECKER]
  DoD: checker runs a signature pass before body analysis.

### Phase 02: Nominal Identity and Equality [W05-P02-NOMINAL-IDENTITY-AND-EQUALITY]

- Task 01: Implement nominal type equality [W05-P02-T01-NOMINAL-EQUALITY]
  DoD: same-name types from different modules are distinct.
- Task 02: Register primitive method surface [W05-P02-T02-PRIMITIVE-METHODS]
  DoD: primitive method calls resolve during body checking.
- Task 03: Implement numeric widening rules [W05-P02-T03-NUMERIC-WIDENING]
  DoD: legal mixed-width arithmetic and comparisons type-check.

### Phase 03: Trait Resolution and Bound Dispatch [W05-P03-TRAIT-RESOLUTION]

- Task 01: Implement trait method lookup on concrete types
  [W05-P03-T01-CONCRETE-TRAIT-METHOD-LOOKUP]
  DoD: trait-implemented methods resolve on concrete receivers.
- Task 02: Implement bound-chain lookup on type parameters
  [W05-P03-T02-BOUND-CHAIN-LOOKUP]
  DoD: bounds and supertraits are searched recursively.
- Task 03: Support trait-typed parameters as interfaces
  [W05-P03-T03-TRAIT-PARAMETERS-AS-INTERFACES]
  DoD: concrete implementers are accepted at trait-typed call sites.

### Phase 04: Contextual Inference and Literals [W05-P04-CONTEXTUAL-INFERENCE-AND-LITERALS]

- Task 01: Infer generics from expected type [W05-P04-T01-EXPECTED-TYPE-INFERENCE]
  DoD: constructor-style calls infer type args from context.
- Task 02: Handle explicit type args on zero-arg generic calls
  [W05-P04-T02-ZERO-ARG-TYPE-ARGS]
  DoD: no-value-argument generic helpers still specialize correctly.
- Task 03: Normalize literal typing [W05-P04-T03-LITERAL-TYPING]
  DoD: integer and float literals pick contextually valid types when required.

### Phase 05: Stdlib Body Checking [W05-P05-STDLIB-BODY-CHECKING]

- Task 01: Remove stdlib body skips [W05-P05-T01-REMOVE-STDLIB-SKIPS]
  DoD: stdlib and user modules are checked uniformly.
- Task 02: Fix exposed semantic gaps [W05-P05-T02-STDLIB-STRESS-FIXES]
  DoD: stdlib methods, traits, and patterns type-check without workarounds.
- Task 03: Add checker regression corpus [W05-P05-T03-CHECKER-REGRESSIONS]
  DoD: each fixed checker class has a dedicated regression.

## Wave 06: Ownership, Liveness, and Destruction

Goal: compute ownership and liveness once and expose it as stable metadata for
all later passes.

Entry criterion: Wave 05 done.

Exit criteria:

- ownership metadata is complete on HIR
- liveness is computed exactly once per function
- destruction behavior is inserted based on last use and ownership

### Phase 01: Ownership Semantics [W06-P01-OWNERSHIP-SEMANTICS]

- Task 01: Model ownership contexts [W06-P01-T01-OWNERSHIP-CONTEXTS]
  DoD: value, ref, mutref, owned, and move contexts are tracked explicitly.
- Task 02: Enforce implicit and explicit borrow rules
  [W06-P01-T02-BORROW-RULES]
  DoD: mutable-receiver implicit borrow and invalid escapes are both enforced.

### Phase 02: Single Liveness Computation [W06-P02-SINGLE-LIVENESS-COMPUTATION]

- Task 01: Compute per-node live-after data [W06-P02-T01-LIVE-AFTER]
  DoD: HIR nodes carry live-after metadata.
- Task 02: Expose last-use and destroy-after metadata
  [W06-P02-T02-LAST-USE-AND-DESTROY-AFTER]
  DoD: later passes do not need to recompute liveness.

### Phase 03: Deterministic Destruction [W06-P03-DETERMINISTIC-DESTRUCTION]

- Task 01: Insert drop intent semantically [W06-P03-T01-DROP-INTENT]
  DoD: owned locals have deterministic destruction behavior on all paths.
- Task 02: Test loops, breaks, and early returns
  [W06-P03-T02-CONTROL-FLOW-DESTRUCTION]
  DoD: destruction remains correct across complex control flow.

## Wave 07: HIR to MIR Lowering

Goal: lower semantically complete HIR into explicit MIR without losing type,
ownership, or control-flow invariants.

Entry criterion: Wave 06 done.

Exit criteria:

- MIR blocks terminate structurally
- no move-after-move invariant violations exist
- method calls and field accesses are disambiguated correctly
- diverging control flow does not create phantom locals

### Phase 01: MIR Core [W07-P01-MIR-CORE]

- Task 01: Define MIR instruction set [W07-P01-T01-MIR-INSTRS]
  DoD: borrow, move, drop, call, field, and constant operations are explicit.
- Task 02: Define MIR builders [W07-P01-T02-MIR-BUILDERS]
  DoD: block and local construction is centralized.

### Phase 02: Control Flow Lowering [W07-P02-CONTROL-FLOW-LOWERING]

- Task 01: Lower branching and loops [W07-P02-T01-BRANCHES-AND-LOOPS]
  DoD: joins exist only when control flow truly reaches them.
- Task 02: Seal blocks on terminators [W07-P02-T02-SEAL-BLOCKS]
  DoD: `return`, `break`, and `continue` do not reopen fallthrough blocks.
- Task 03: Model divergence structurally [W07-P02-T03-DIVERGENCE-STRUCTURE]
  DoD: no fake post-divergence temporaries appear in MIR.

### Phase 03: Calls, Methods, and Fields [W07-P03-CALLS-METHODS-AND-FIELDS]

- Task 01: Lower borrow expressions as borrow instructions
  [W07-P03-T01-BORROW-INSTRS]
  DoD: `ref` and `mutref` never use generic unary lowering.
- Task 02: Disambiguate method calls from field reads
  [W07-P03-T02-METHOD-VS-FIELD]
  DoD: method calls do not lower to field-address instructions.
- Task 03: Lower enum constructors and bare variant values
  [W07-P03-T03-ENUM-CONSTRUCTORS]
  DoD: enum variant values and constructor calls lower distinctly and correctly.

## Wave 08: Runtime Library

Goal: implement the bootstrap runtime surface in C with a stable ABI and a small
trusted code footprint.

Entry criterion: Wave 00 done.

Exit criteria:

- all required runtime entry points exist
- runtime tests pass on supported host platforms
- runtime ABI matches the language guide and backend contracts

### Phase 01: Runtime Surface [W08-P01-RUNTIME-SURFACE]

- Task 01: Define runtime header [W08-P01-T01-RUNTIME-HEADER]
  DoD: all bootstrap runtime entry points are declared.
- Task 02: Implement memory and panic primitives
  [W08-P01-T02-MEMORY-AND-PANIC]
  DoD: allocation, deallocation, panic, and abort surface exist.

### Phase 02: IO, Process, and Time [W08-P02-IO-PROCESS-AND-TIME]

- Task 01: Implement basic IO surface [W08-P02-T01-BASIC-IO]
  DoD: stdout, stderr, file, and minimal path operations work.
- Task 02: Implement process and time surface [W08-P02-T02-PROCESS-AND-TIME]
  DoD: arguments, environment, clock, and process control work.

### Phase 03: Threads and Synchronization [W08-P03-THREADS-AND-SYNC]

- Task 01: Implement thread and TLS surface [W08-P03-T01-THREAD-AND-TLS]
  DoD: spawn and thread-local primitives work.
- Task 02: Implement synchronization surface [W08-P03-T02-SYNC]
  DoD: mutex, condition, and related runtime helpers work.

## Wave 09: C11 Backend and Representation Contracts

Goal: emit correct, deterministic C11 from concrete MIR while enforcing the
backend contracts documented in the language guide.

Entry criterion: Waves 07 and 08 done.

Exit criteria:

- composite types are emitted before use
- unresolved types never reach codegen
- pointer categories are handled correctly
- unit erasure is total
- divergence and aggregate fallbacks are emitted correctly
- a hello-world program builds and runs through the full pipeline

### Phase 01: Type Emission and Naming [W09-P01-TYPE-EMISSION-AND-NAMING]

- Task 01: Emit composite type definitions before function bodies
  [W09-P01-T01-TYPE-DEFS-FIRST]
  DoD: generated C has no use-before-definition composite types.
- Task 02: Sanitize identifiers and avoid collisions
  [W09-P01-T02-IDENTIFIER-SANITIZATION]
  DoD: names are legal C and stable across builds.
- Task 03: Encode module-qualified identity in mangling
  [W09-P01-T03-MODULE-QUALIFIED-MANGLING]
  DoD: same-name items from different modules do not collide.

### Phase 02: Pointer Categories and Borrow Semantics [W09-P02-POINTER-CATEGORIES-AND-BORROWS]

- Task 01: Distinguish borrow pointers from `Ptr[T]` values
  [W09-P02-T01-TWO-POINTER-CATEGORIES]
  DoD: codegen treats the two categories differently.
- Task 02: Adapt call sites to ref and mutref signatures
  [W09-P02-T02-CALL-SITE-ADAPTATION]
  DoD: value and borrow arguments are passed correctly.

### Phase 03: Unit Erasure and Aggregate Emission [W09-P03-UNIT-ERASURE-AND-AGGREGATE-EMISSION]

- Task 01: Erase unit consistently [W09-P03-T01-TOTAL-UNIT-ERASURE]
  DoD: no ghost unit payloads or params remain in generated C.
- Task 02: Emit typed aggregate zero-initializers
  [W09-P03-T02-TYPED-AGGREGATE-FALLBACKS]
  DoD: aggregate fallbacks are never scalar `0`.

### Phase 04: Divergence and Equality Lowering [W09-P04-DIVERGENCE-AND-EQUALITY-LOWERING]

- Task 01: Emit divergence structurally [W09-P04-T01-STRUCTURAL-DIVERGENCE]
  DoD: no undeclared locals are read after diverging calls.
- Task 02: Lower equality and comparison semantically
  [W09-P04-T02-SEMANTIC-EQUALITY]
  DoD: non-scalar equality does not compile down to invalid raw C comparisons.

### Phase 05: Backend Regression Closure [W09-P05-BACKEND-REGRESSION-CLOSURE]

- Task 01: Add regression tests from fuse3 bug history
  [W09-P05-T01-BUG-HISTORY-REGRESSIONS]
  DoD: codegen bugs discovered in fuse3 have direct regression coverage.

## Wave 10: Native Build Driver and Linking

Goal: turn generated C and the runtime library into working native artifacts.

Entry criterion: Wave 09 done.

Exit criteria:

- `fuse build` produces working binaries and libraries
- runtime library discovery is deterministic
- build errors are rendered clearly

### Phase 01: Compiler Invocation [W10-P01-COMPILER-INVOCATION]

- Task 01: Detect host C compiler [W10-P01-T01-COMPILER-DETECTION]
  DoD: supported host toolchains are discovered reliably.
- Task 02: Emit compile and link arguments [W10-P01-T02-COMPILE-LINK-ARGS]
  DoD: target, optimization, debug, and output flags work.

### Phase 02: Runtime Discovery and Linking [W10-P02-RUNTIME-DISCOVERY-AND-LINKING]

- Task 01: Discover or build runtime library [W10-P02-T01-RUNTIME-DISCOVERY]
  DoD: package builds find the runtime deterministically.
- Task 02: Link artifacts [W10-P02-T02-LINK-ARTIFACTS]
  DoD: executables and supported library outputs link correctly.

### Phase 03: End-to-End Validation [W10-P03-END-TO-END-VALIDATION]

- Task 01: Build and run examples [W10-P03-T01-BUILD-AND-RUN-EXAMPLES]
  DoD: hello, echo, and representative examples pass.
- Task 02: Add end-to-end failure diagnostics
  [W10-P03-T02-E2E-DIAGNOSTICS]
  DoD: build errors point to generated C only as implementation failures.

## Wave 11: CLI, Diagnostics, and Developer Workflows

Goal: expose the compiler through a coherent command-line interface and stable
developer workflow.

Entry criterion: Wave 10 done.

Exit criteria:

- build, run, check, test, fmt, doc, repl, version, and help flows exist
- diagnostics are readable in text and JSON
- tooling behavior is deterministic

### Phase 01: CLI Surface [W11-P01-CLI-SURFACE]

- Task 01: Implement subcommand parser [W11-P01-T01-SUBCOMMAND-PARSER]
  DoD: common flags and subcommand-specific flags work.
- Task 02: Wire top-level commands [W11-P01-T02-COMMAND-WIRING]
  DoD: build, run, check, test, fmt, doc, repl, version, and help dispatch.

### Phase 02: Diagnostics and Formatting [W11-P02-DIAGNOSTICS-AND-FORMATTING]

- Task 01: Stabilize diagnostic rendering [W11-P02-T01-DIAGNOSTIC-RENDERING]
  DoD: human-readable diagnostics include spans and context.
- Task 02: Add JSON output mode [W11-P02-T02-JSON-DIAGNOSTICS]
  DoD: machine-readable diagnostics are emitted consistently.

### Phase 03: Workflow Tools [W11-P03-WORKFLOW-TOOLS]

- Task 01: Implement doc and format workflows [W11-P03-T01-DOC-AND-FMT]
  DoD: public API docs and formatting workflows are usable.
- Task 02: Implement REPL and test runner workflows
  [W11-P03-T02-REPL-AND-TESTRUNNER]
  DoD: developer-facing tools run against the same compiler core.

## Wave 12: Core Standard Library

Goal: implement the OS-free core standard library and use it as a semantic and
backend stress suite.

Entry criterion: Wave 11 done.

Exit criteria:

- core traits, primitives, strings, collections, iterators, and formatting ship
- all core public APIs are documented
- core library passes tests and compiler stress cases

### Phase 01: Core Traits and Primitive Surface [W12-P01-CORE-TRAITS-AND-PRIMITIVES]

- Task 01: Ship core traits [W12-P01-T01-CORE-TRAITS]
  DoD: equality, hashing, comparison, formatting, and default traits exist.
- Task 02: Implement primitive methods and aliases
  [W12-P01-T02-PRIMITIVE-METHOD-SURFACE]
  DoD: integer, float, bool, and char methods match language contracts.

### Phase 02: Strings, Collections, and Iteration [W12-P02-STRINGS-COLLECTIONS-AND-ITERATION]

- Task 01: Implement String and formatting primitives
  [W12-P02-T01-STRING-AND-FMT]
  DoD: `String` and formatter builders are usable.
- Task 02: Implement List, Map, Set, and iterators
  [W12-P02-T02-COLLECTIONS-AND-ITERATORS]
  DoD: collections stress generics, traits, and ownership correctly.

### Phase 03: Runtime Bridge Layer [W12-P03-RUNTIME-BRIDGE-LAYER]

- Task 01: Implement core bridge files [W12-P03-T01-CORE-BRIDGE-FILES]
  DoD: only approved bridge files contain unsafe runtime hooks.
- Task 02: Add doc coverage checks [W12-P03-T02-DOC-COVERAGE-CHECKS]
  DoD: all public stdlib APIs have docs.

## Wave 13: Hosted Standard Library

Goal: implement the hosted stdlib tier on top of core while preserving the core
versus hosted boundary.

Entry criterion: Wave 12 done.

Exit criteria:

- full tier IO, fs, os, time, thread, sync, and chan modules exist
- concurrency surface passes threaded tests
- hosted modules do not leak back into core

### Phase 01: IO and OS Surface [W13-P01-IO-AND-OS-SURFACE]

- Task 01: Implement IO modules [W13-P01-T01-IO-MODULES]
  DoD: stdin, stdout, stderr, files, and dirs work.
- Task 02: Implement OS modules [W13-P01-T02-OS-MODULES]
  DoD: env, process, args, and time modules work.

### Phase 02: Threads, Sync, and Channels [W13-P02-THREADS-SYNC-AND-CHANNELS]

- Task 01: Implement thread and handle modules
  [W13-P02-T01-THREAD-AND-HANDLE]
  DoD: spawn and thread handles work.
- Task 02: Implement sync modules [W13-P02-T02-SYNC-MODULES]
  DoD: mutex, rwlock, cond, once, and shared APIs work.
- Task 03: Implement channels [W13-P02-T03-CHANNELS]
  DoD: channel operations reflect the concurrency model from the guide.

## Wave 14: Stage 2 Compiler Port

Goal: bring up the self-hosted Fuse compiler source tree until it builds cleanly
with Stage 1.

Entry criterion: Wave 13 done.

Exit criteria:

- stage2 source mirrors stage1 architecture closely enough to bootstrap
- stage1 compiles stage2 end to end
- stage2 build failures are reduced to real stage2 defects, not stage1 gaps

### Phase 01: Stage 2 Frontend Parity [W14-P01-STAGE2-FRONTEND-PARITY]

- Task 01: Port frontend modules [W14-P01-T01-PORT-FRONTEND]
  DoD: lex, parse, resolve, HIR, and checker sources exist in Fuse.
- Task 02: Port core driver and helpers [W14-P01-T02-PORT-DRIVER-HELPERS]
  DoD: stage2 has the minimal executable architecture to compile itself.

### Phase 02: Stage 2 Backend Bring-Up [W14-P02-STAGE2-BACKEND-BRING-UP]

- Task 01: Close frontend checker gaps [W14-P02-T01-CLOSE-FRONTEND-GAPS]
  DoD: `fuse check stage2` succeeds.
- Task 02: Close backend contract gaps [W14-P02-T02-CLOSE-BACKEND-GAPS]
  DoD: stage2 generated C compiles significantly beyond example programs.

### Phase 03: Stage 2 Build Closure [W14-P03-STAGE2-BUILD-CLOSURE]

- Task 01: Eliminate stage2 C generation breakpoints
  [W14-P03-T01-ELIMINATE-STAGE2-C-BREAKPOINTS]
  DoD: no remaining failures are explained by missing stage1 contracts.
- Task 02: Build stage2 end to end [W14-P03-T02-BUILD-STAGE2-END-TO-END]
  DoD: stage1 produces a working stage2 compiler artifact.

## Wave 15: Self-Hosting Gate

Goal: prove that Fuse can compile itself reproducibly.

Entry criterion: Wave 14 done.

Exit criteria:

- stage1 compiles stage2 successfully
- stage2 recompiles itself successfully
- output equivalence or reproducibility checks pass according to project policy

### Phase 01: First Self-Compilation [W15-P01-FIRST-SELF-COMPILATION]

- Task 01: Compile stage2 with stage1 [W15-P01-T01-STAGE1-COMPILES-STAGE2]
  DoD: a working stage2 artifact is produced.
- Task 02: Compile stage2 with stage2 [W15-P01-T02-STAGE2-COMPILES-ITSELF]
  DoD: self-compilation completes successfully.

### Phase 02: Reproducibility and Equivalence [W15-P02-REPRODUCIBILITY-AND-EQUIVALENCE]

- Task 01: Implement bootstrap reproducibility check
  [W15-P02-T01-BOOTSTRAP-REPRO-CHECK]
  DoD: multi-generation outputs compare according to policy.
- Task 02: Gate merges on bootstrap health [W15-P02-T02-GATE-ON-BOOTSTRAP]
  DoD: self-hosting regressions are release-blocking.

## Wave 16: Native Backend Transition

Goal: remove the bootstrap C11 backend from the compiler implementation path by
introducing the native backend on top of the same semantic contracts.

Entry criterion: Wave 15 done.

Exit criteria:

- native backend exists and passes correctness gates
- stage2 no longer depends on C11 codegen for its own build path

### Phase 01: Native Backend Foundation [W16-P01-NATIVE-BACKEND-FOUNDATION]

- Task 01: Define native backend interface [W16-P01-T01-NATIVE-BACKEND-INTERFACE]
  DoD: native backend consumes MIR without C11-specific assumptions.
- Task 02: Reuse backend contracts [W16-P01-T02-REUSE-BACKEND-CONTRACTS]
  DoD: pointer, unit, monomorphization, and divergence contracts remain intact.

### Phase 02: Native Backend Closure [W16-P02-NATIVE-BACKEND-CLOSURE]

- Task 01: Compile stage2 through native path [W16-P02-T01-COMPILE-STAGE2-NATIVELY]
  DoD: stage2 builds without C11 backend dependency.
- Task 02: Remove bootstrap-only C11 requirement
  [W16-P02-T02-REMOVE-C11-REQUIREMENT]
  DoD: C11 backend becomes optional or retired.

## Wave 17: Retirement of Go and C from the Compiler Path

Goal: complete the transition from bootstrap implementation languages to a Fuse
compiler implemented and built by Fuse.

Entry criterion: Wave 16 done.

Exit criteria:

- Fuse owns the compiler implementation path
- Go is no longer required to build the compiler
- C is no longer required as a backend or runtime implementation dependency in
  the compiler path

### Phase 01: Retire Go [W17-P01-RETIRE-GO]

- Task 01: Freeze Stage 1 as archival bootstrap tool [W17-P01-T01-FREEZE-STAGE1]
  DoD: Stage 1 is no longer required for ordinary compiler development.
- Task 02: Remove Go from active compiler build workflow
  [W17-P01-T02-REMOVE-GO-FROM-WORKFLOW]
  DoD: supported build path no longer invokes Go.

### Phase 02: Retire C [W17-P02-RETIRE-C]

- Task 01: Replace C runtime dependencies as required
  [W17-P02-T01-REPLACE-C-RUNTIME]
  DoD: compiler implementation path no longer requires C runtime code.
- Task 02: Remove C from compiler bootstrap assumptions
  [W17-P02-T02-REMOVE-C-FROM-BOOTSTRAP-ASSUMPTIONS]
  DoD: compiler implementation is Fuse-only.

## Wave 18: Targets and Ecosystem Growth

Goal: resume broader target and library work on top of the self-hosted native
compiler.

Entry criterion: Wave 17 done.

Exit criteria:

- target expansion and library growth occur without reintroducing bootstrap debt

### Phase 01: Additional Targets [W18-P01-ADDITIONAL-TARGETS]

- Task 01: Add target descriptions [W18-P01-T01-TARGET-DESCRIPTIONS]
  DoD: each supported target has a documented ABI and validation path.
- Task 02: Add target CI [W18-P01-T02-TARGET-CI]
  DoD: target regressions are visible immediately.

### Phase 02: Extended Libraries [W18-P02-EXTENDED-LIBRARIES]

- Task 01: Implement ext stdlib modules [W18-P02-T01-EXT-STDLIB]
  DoD: optional libraries build on the stable core and hosted tiers.
- Task 02: Publish ecosystem guidance [W18-P02-T02-ECOSYSTEM-GUIDANCE]
  DoD: package authors have clear compatibility and safety rules.

## Cross-cutting constraints

The following rules apply to every wave.

- Determinism is a release-level requirement.
- No unresolved types may reach codegen.
- No pass may recompute liveness independently.
- Invariant walkers remain enabled in debug and CI contexts.
- Stdlib failures are compiler signals, not library excuses.
- Workarounds are forbidden.
- Each non-trivial bug must produce both a regression and a learning-log entry.