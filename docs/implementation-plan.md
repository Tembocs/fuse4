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
| 17 | Generics end-to-end | Wave 16 done | generic functions and types compile, run, and produce correct output |
| 18 | Language completeness | Wave 17 done | every language-guide feature has an e2e proof program; stdlib compiles and runs |
| 19 | Retirement of Go and C | Wave 18 done | Fuse owns the compiler implementation path |
| 20 | Targets and ecosystem | Wave 19 done | cross-target and library growth resume on native base |

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

### Phase 06: Monomorphization [W05-P06-MONOMORPHIZATION]

- Task 01: Collect concrete instantiations at call sites
  [W05-P06-T01-COLLECT-INSTANTIATIONS]
  DoD: all generic function and type usages produce concrete type argument sets.
- Task 02: Validate specialization completeness
  [W05-P06-T02-VALIDATE-SPECIALIZATIONS]
  DoD: partially-resolved type arguments are rejected before lowering.
- Task 03: Specialize generic functions into concrete HIR/MIR
  [W05-P06-T03-SPECIALIZE-FUNCTIONS]
  DoD: each concrete instantiation produces a distinct lowered function.
- Task 04: Integrate monomorph into the driver pipeline
  [W05-P06-T04-INTEGRATE-PIPELINE]
  DoD: `Build()` runs monomorphization between checking and lowering.

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

### Phase 04: Pattern Match Lowering [W07-P04-PATTERN-MATCH-LOWERING]

- Task 01: Add structured pattern nodes to HIR
  [W07-P04-T01-HIR-PATTERN-NODES]
  DoD: MatchArm carries structured Pattern (LiteralPat, BindPat,
  ConstructorPat, WildcardPat), not a text description.
- Task 02: Lower match to cascading branches
  [W07-P04-T02-MATCH-TO-BRANCHES]
  DoD: each arm produces a condition check and branch; wildcard arms produce
  unconditional jumps; arms are tested in declaration order.
- Task 03: Lower enum discriminant access
  [W07-P04-T03-ENUM-DISCRIMINANT-ACCESS]
  DoD: match on enum types reads the tag field and branches by tag value.

### Phase 05: Error Propagation Lowering [W07-P05-ERROR-PROPAGATION-LOWERING]

- Task 01: Type-check ? operator on Result and Option
  [W07-P05-T01-QUESTION-TYPECHECK]
  DoD: `?` on `Result[T, E]` returns `T`; `?` on `Option[T]` returns `T`;
  type errors are diagnosed.
- Task 02: Lower ? to branch-and-early-return
  [W07-P05-T02-QUESTION-LOWERING]
  DoD: `?` emits a discriminant check, extracts Ok/Some value on success,
  and returns Err/None on failure.

### Phase 06: Closure Lowering [W07-P06-CLOSURE-LOWERING]

- Task 01: Implement capture analysis
  [W07-P06-T01-CAPTURE-ANALYSIS]
  DoD: closure bodies are scanned for references to outer variables.
- Task 02: Generate environment struct and lift closure body
  [W07-P06-T02-CLOSURE-LIFTING]
  DoD: each closure produces a lifted function taking an env parameter and
  a struct type holding captured variables.
- Task 03: Emit closure construction at expression site
  [W07-P06-T03-CLOSURE-CONSTRUCTION]
  DoD: closure expressions emit struct init for the environment and pair it
  with the lifted function pointer.

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

### Phase 05: Drop Codegen [W09-P05-DROP-CODEGEN]

- Task 01: Flow Drop trait metadata to codegen
  [W09-P05-T01-DROP-TRAIT-METADATA]
  DoD: codegen can determine whether a type has a Drop implementation.
- Task 02: Emit destructor calls for InstrDrop
  [W09-P05-T02-EMIT-DESTRUCTOR-CALLS]
  DoD: `InstrDrop` on types with Drop impls emits `TypeName_drop(&_lN);`
  in generated C. Types without Drop emit no-ops.
- Task 03: Test drop codegen with owned resources
  [W09-P05-T03-DROP-CODEGEN-TESTS]
  DoD: regression tests verify destructor calls appear in generated C.

### Phase 06: Backend Regression Closure [W09-P06-BACKEND-REGRESSION-CLOSURE]

- Task 01: Add regression tests from fuse3 bug history
  [W09-P06-T01-BUG-HISTORY-REGRESSIONS]
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

### Phase 03: Compiler-Side Concurrency Integration [W13-P03-COMPILER-CONCURRENCY-INTEGRATION]

- Task 01: Add channel type kind to the type table
  [W13-P03-T01-CHANNEL-TYPE-KIND]
  DoD: `KindChannel` exists in the type table with an element type parameter.
- Task 02: Lower spawn to runtime thread creation
  [W13-P03-T02-SPAWN-TO-RUNTIME]
  DoD: `spawn expr` emits `fuse_rt_thread_spawn(fn, arg)` in codegen.
- Task 03: Lower channel operations to runtime calls
  [W13-P03-T03-CHANNEL-OPS-TO-RUNTIME]
  DoD: send, recv, and close emit corresponding `fuse_rt_*` calls.
- Task 04: Type-check channel expressions with element types
  [W13-P03-T04-CHANNEL-TYPECHECK]
  DoD: `Chan[I32]` is a valid type; send/recv are type-checked against the
  channel element type.

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

## Wave 17: Generics End-to-End

Goal: make generic functions and generic types compile through the full pipeline
and produce correct running programs. Currently the `monomorph` package exists
but is not integrated, the checker resolves generic type args but does not
propagate them to specialization, and no generic program has ever compiled to a
working binary.

Entry criterion: Wave 16 done.

Exit criteria:

- a generic function called with two different concrete types produces two
  distinct specialized functions in the generated C
- `Option[I32]` and `Result[I32, Bool]` compile and run through pattern matching
- all proof programs in this wave compile, link, run, and return the expected
  exit code

Proof programs (each must pass as an e2e test):

```fuse
// P1: basic generic function
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity[I32](42); }
// expected: exit code 42

// P2: two instantiations of the same generic
fn first[T](a: T, b: T) -> T { return a; }
fn main() -> I32 {
    let x = first[I32](10, 20);
    let y = first[I32](3, 7);
    return x + y;
}
// expected: exit code 13

// P3: generic type (Option) with pattern matching
// (requires enum layout, discriminant, and match dispatch)
```

### Phase 01: Generic Parameter Scoping in the Checker [W17-P01-GENERIC-PARAM-SCOPING]

Currently the checker resolves type expressions but does not register generic
type parameters (`T`, `U`) as in-scope types during body checking of generic
functions. This phase ensures `T` is a valid type inside the body.

- Task 01: Register generic params in the function's local type scope
  [W17-P01-T01-REGISTER-GENERIC-PARAMS]
  Currently stubbed: the checker ignores `fn.GenericParams` during body
  checking (`compiler/check/checker.go:208-224`).
  DoD: inside `fn identity[T](x: T) -> T { return x; }`, the parameter `x`
  resolves to type `T` (a GenericParam TypeId), and the return type resolves to
  `T`.
- Task 02: Resolve explicit type arguments at call sites
  [W17-P01-T02-RESOLVE-CALL-SITE-TYPE-ARGS]
  Currently stubbed: `checkCall` does not match explicit type args like
  `identity[I32](42)` to the generic function's type parameters.
  DoD: `identity[I32](42)` resolves `T=I32` and the call's return type is
  `I32`.
- Task 03: Add checker tests for generic param scoping
  [W17-P01-T03-CHECKER-GENERIC-TESTS]
  DoD: unit tests verify that generic param types flow through function
  bodies and that explicit type args at call sites resolve correctly.

### Phase 02: Instantiation Collection [W17-P02-INSTANTIATION-COLLECTION]

The `compiler/monomorph/` package has `Context.Record()` but it is never called.
This phase integrates it into the driver pipeline.

- Task 01: Scan checked AST for generic call sites
  [W17-P02-T01-SCAN-GENERIC-CALL-SITES]
  Currently missing: no code walks the checked AST to find calls with type
  arguments.
  DoD: after checking, the driver collects all concrete instantiations
  (function name + type arg list) via `monomorph.Context.Record()`.
- Task 02: Validate instantiation completeness
  [W17-P02-T02-VALIDATE-INSTANTIATION-COMPLETENESS]
  Currently implemented in `monomorph.Record()` (rejects partial
  specializations). Verify it works in the integrated pipeline.
  DoD: a call like `identity[Unknown]()` produces a diagnostic, not a silent
  fallback.
- Task 03: Add driver tests for instantiation collection
  [W17-P02-T03-INSTANTIATION-COLLECTION-TESTS]
  DoD: driver tests verify that `Build()` with a generic program produces
  the expected number of instantiations and rejects incomplete ones.

### Phase 03: Generic Function Body Specialization [W17-P03-BODY-SPECIALIZATION]

This is the core of monomorphization: duplicating a generic function body with
concrete types substituted for type parameters.

- Task 01: Implement AST-level body duplication with type substitution
  [W17-P03-T01-AST-BODY-DUPLICATION]
  Currently missing: no code duplicates a function body.
  DoD: given `fn identity[T](x: T) -> T { return x; }` and `T=I32`, a new
  concrete `FnDecl` is produced with `T` replaced by `I32` in all parameter
  types, return types, and body type expressions.
- Task 02: Generate specialized function names
  [W17-P03-T02-SPECIALIZED-FUNCTION-NAMES]
  Currently missing: no naming scheme for specialized functions.
  DoD: `identity[I32]` produces a function named `identity__I32` (or similar
  deterministic scheme). The name is used consistently in declaration and at
  call sites.
- Task 03: Rewrite call sites to reference specialized names
  [W17-P03-T03-REWRITE-CALL-SITES]
  Currently missing: call sites reference the generic name.
  DoD: `identity[I32](42)` in the caller's body is rewritten to call
  `identity__I32(42)`.
- Task 04: Add unit tests for body specialization
  [W17-P03-T04-SPECIALIZATION-TESTS]
  DoD: unit tests verify that a generic function produces a correctly typed
  concrete function after substitution, and that call sites reference the
  specialized name.

### Phase 04: Driver Pipeline Integration [W17-P04-DRIVER-PIPELINE-INTEGRATION]

Wire the specialization output into the existing driver pipeline so specialized
functions flow through HIR → liveness → MIR → codegen.

- Task 01: Insert specialization step between checking and HIR building
  [W17-P04-T01-INSERT-SPECIALIZATION-STEP]
  Currently missing: the driver goes directly from checking to HIR building.
  DoD: the driver runs instantiation collection, then body specialization,
  then feeds the specialized concrete functions (not the generic originals)
  into the HIR builder.
- Task 02: Skip generic function originals in codegen
  [W17-P04-T02-SKIP-GENERIC-ORIGINALS]
  Currently missing: generic functions with unresolved type params would
  produce invalid C if lowered.
  DoD: functions with `GenericParams` are not lowered unless they have been
  specialized. Only concrete specializations reach codegen.
- Task 03: Verify generated C for basic generic proof program
  [W17-P04-T03-VERIFY-GENERATED-C]
  DoD: `identity[I32](42)` produces C containing a function
  `Fuse_identity__I32` that takes `int32_t` and returns `int32_t`, and
  `main` calls it. The C compiles with gcc.

### Phase 05: Proof Program P1 — Basic Generic Function [W17-P05-PROOF-P1]

- Task 01: Add e2e test for basic generic function
  [W17-P05-T01-E2E-BASIC-GENERIC]
  DoD: the following program compiles, runs, and exits with code 42:
  `fn identity[T](x: T) -> T { return x; } fn main() -> I32 { return identity[I32](42); }`
- Task 02: Fix any failures surfaced by the proof program
  [W17-P05-T02-FIX-P1-FAILURES]
  DoD: the e2e test passes. Any bugs found are fixed with regressions and
  learning-log entries.

### Phase 06: Multiple Instantiations [W17-P06-MULTIPLE-INSTANTIATIONS]

- Task 01: Verify two instantiations of the same generic produce distinct
  functions [W17-P06-T01-TWO-INSTANTIATIONS]
  Currently: `monomorph.Record()` deduplicates, so `identity[I32]` and
  `identity[Bool]` should produce two entries.
  DoD: generated C contains both `Fuse_identity__I32` and
  `Fuse_identity__Bool`.
- Task 02: Add e2e test for multiple instantiations
  [W17-P06-T02-E2E-MULTIPLE-INSTANTIATIONS]
  DoD: a program calling `first[I32](10, 20)` and using the result compiles,
  runs, and exits with the correct code.

### Phase 07: Generic Types (Option, Result) [W17-P07-GENERIC-TYPES]

Generic types like `Option[T]` require enum layout with a discriminant tag
and payload fields that vary by type argument. This phase extends
monomorphization from functions to types.

- Task 01: Specialize generic enum types
  [W17-P07-T01-SPECIALIZE-GENERIC-ENUMS]
  Currently missing: `Option[I32]` is interned as a TypeId with TypeArgs but
  no concrete enum layout is generated.
  DoD: `Option[I32]` produces a concrete C struct with a `_tag` field and
  a payload field of type `int32_t`.
- Task 02: Emit specialized enum type definitions in codegen
  [W17-P07-T02-EMIT-SPECIALIZED-ENUM-TYPES]
  Currently: codegen emits `typedef struct Name Name;` for enums but does not
  emit field definitions for the tag and payload.
  DoD: generated C for `Option[I32]` includes a struct definition with `int
  _tag;` and `int32_t _f0;` fields.
- Task 03: Specialize generic struct types
  [W17-P07-T03-SPECIALIZE-GENERIC-STRUCTS]
  DoD: generic structs with type parameters produce concrete field layouts
  after substitution.

### Phase 08: Enum Construction and Destructuring with Generics [W17-P08-ENUM-GENERICS]

- Task 01: Emit specialized enum variant constructors
  [W17-P08-T01-SPECIALIZED-ENUM-CONSTRUCTORS]
  DoD: `Option.Some[I32](42)` produces C that initializes the specialized
  Option_I32 struct with `_tag=0` and `_f0=42`.
- Task 02: Lower match on specialized enums to discriminant dispatch
  [W17-P08-T02-MATCH-ON-SPECIALIZED-ENUMS]
  DoD: `match opt { Some(v) { ... } None { ... } }` on `Option[I32]` reads
  `_tag`, branches, and extracts `_f0` as `int32_t`.
- Task 03: Add e2e test for Option with pattern matching
  [W17-P08-T03-E2E-OPTION-MATCH]
  DoD: a program that creates `Some(42)`, matches on it, and returns the
  inner value compiles, runs, and exits with code 42.

### Phase 09: Error Propagation with Generics [W17-P09-ERROR-PROPAGATION-GENERICS]

The `?` operator was implemented for the lowerer (L009) but requires a
specialized `Result[T, E]` type to work end-to-end.

- Task 01: Verify `?` on specialized Result type
  [W17-P09-T01-QUESTION-ON-SPECIALIZED-RESULT]
  DoD: `?` on `Result[I32, Bool]` reads `_tag`, branches, extracts `_f0` as
  `int32_t` on success, and early-returns the Result on failure.
- Task 02: Add e2e test for error propagation
  [W17-P09-T02-E2E-ERROR-PROPAGATION]
  DoD: a program that uses `?` on a Result value compiles, runs, and returns
  the expected exit code for both the Ok and Err paths.

### Phase 10: Regression Closure [W17-P10-REGRESSION-CLOSURE]

- Task 01: Add regression tests for all fixed generic edge cases
  [W17-P10-T01-GENERIC-REGRESSIONS]
  DoD: every bug found during this wave has a regression test.
- Task 02: Update learning log with any new lessons
  [W17-P10-T02-UPDATE-LEARNING-LOG]
  DoD: learning-log entries exist for any design gaps or architectural
  lessons discovered during generic implementation.
- Task 03: Verify all proof programs pass in CI
  [W17-P10-T03-CI-PROOF-PROGRAMS]
  DoD: the e2e test suite including all generic proof programs passes in the
  CI matrix (Linux, macOS, Windows).

## Wave 18: Language Completeness

Goal: close every feature gap between the language guide and the compiler so
that the full standard library compiles, every documented feature has an e2e
proof program, and a user can write non-trivial Fuse programs that use
closures, I/O, iteration, traits, and concurrency.

This wave was added after learning-log entry L016 identified 40 feature gaps
across 4 categories: implemented-but-unproven, partially implemented,
specified-but-not-implemented, and stdlib gaps. Every item in L016 must be
resolved before Wave 19 (Retirement of Go and C) can begin.

Entry criterion: Wave 17 done.

Exit criteria:

- every feature in the language guide (sections 1–17) has at least one e2e
  proof program that compiles, links, runs, and produces verified output
- `stdlib/core/` compiles through the pipeline with no type errors
- `stdlib/full/` compiles through the pipeline with no type errors
- `print("hello")` and `println("world")` produce correct stdout output
- `for x in iter { ... }` works with the Iterator trait
- closures capture outer variables and execute correctly
- trait method dispatch on concrete types works end-to-end
- Drop destructors run at scope exit for types with Drop implementations
- channels and spawn produce concurrent behavior
- all proof programs pass in the CI matrix

Proof programs (each must pass as an e2e test):

```fuse
// P1: closures
fn apply(f: Fn(I32) -> I32, x: I32) -> I32 { return f(x); }
fn main() -> I32 {
    let offset = 10;
    let f = |x| { x + offset };
    return apply(f, 32);
}
// expected: exit code 42

// P2: trait dispatch
trait Doubler { fn double(ref self) -> I32; }
impl Doubler for I32 { fn double(ref self) -> I32 { return self * 2; } }
fn do_double(x: ref Doubler) -> I32 { return x.double(); }
fn main() -> I32 { let v: I32 = 21; return do_double(ref v); }
// expected: exit code 42

// P3: for..in iteration
fn main() -> I32 {
    var sum = 0;
    for x in [1, 2, 3, 4] { sum = sum + x; }
    return sum;
}
// expected: exit code 10

// P4: Drop destructor
// (verify destructor runs by setting a global flag or exit code)

// P5: print/println to stdout
fn main() -> I32 { println("hello"); return 0; }
// expected: stdout contains "hello", exit code 0

// P6: channels and spawn
fn main() -> I32 {
    let ch = Chan[I32].new();
    spawn fn() { ch.send(42); };
    match ch.recv() { Ok(v) => return v, Err(_) => return 1 }
}
// expected: exit code 42
```

### Phase 01: Core Expression Completeness [W18-P01-CORE-EXPR-COMPLETENESS]

Features that are implemented in the pipeline but lack e2e proof programs.
Each task produces a proof program that fails if the feature is reverted.

- Task 01: Tuple construction and field access
  [W18-P01-T01-TUPLE-E2E]
  Currently: lowered to InstrTuple, emitted as anonymous struct. No e2e test.
  DoD: `let p = (10, 32); return p.0 + p.1;` compiles, runs, exits 42.

- Task 02: Struct initialization and field access
  [W18-P01-T02-STRUCT-E2E]
  Currently: lowered to InstrStructInit. No e2e test.
  DoD: `struct P { x: I32, y: I32 } ... return p.x + p.y;` exits 42.

- Task 03: Ownership forms — ref, mutref, owned, move
  [W18-P01-T03-OWNERSHIP-E2E]
  Currently: borrow lowering to InstrBorrow with BorrowShared/BorrowMutable
  exists. No e2e test verifies C pointer semantics.
  DoD: program passing `ref` and `mutref` to functions compiles and runs.
  Mutref mutation is visible to the caller.

- Task 04: Loop with break value
  [W18-P01-T04-LOOP-BREAK-VALUE-E2E]
  Currently: lowerer captures break values via BreakLocal. No e2e test.
  DoD: `let x = loop { break 42; }; return x;` exits 42.

- Task 05: Const declarations
  [W18-P01-T05-CONST-DECL]
  Currently: parsed but no const evaluation. Using a const produces Unknown.
  DoD: `const N: I32 = 42; fn main() -> I32 { return N; }` exits 42.

- Task 06: Type aliases
  [W18-P01-T06-TYPE-ALIAS]
  Currently: parsed (`TypeAliasDecl`) but checker never resolves alias to
  underlying type.
  DoD: `type Score = I32; fn main() -> Score { return 42; }` exits 42.

### Phase 02: Closures End-to-End [W18-P02-CLOSURES]

Closures are specified in the language guide (section 5.1) and partially
implemented (capture analysis + function lifting in `compiler/lower/lower.go:
572-648`) but the closure is returned as an environment struct only, not
paired with a function pointer.

- Task 01: Complete closure representation — pair environment with function
  pointer [W18-P02-T01-CLOSURE-REPR]
  Currently: `lowerClosure` returns `envDest` (environment struct) without
  the lifted function pointer (`lower.go:647`).
  DoD: closure values carry both the environment and the function pointer.
  Generated C emits a struct with `{ void* fn; void* env; }` or equivalent.

- Task 02: Implement closure call — invoke lifted function with environment
  [W18-P02-T02-CLOSURE-CALL]
  Currently: calling a closure value would attempt a direct call on the
  environment struct, which is invalid C.
  DoD: `let f = |x| { x + 1 }; return f(41);` compiles and exits 42.

- Task 03: Implement closure capture of outer variables
  [W18-P02-T03-CLOSURE-CAPTURE]
  Currently: capture analysis scans limited expression types.
  DoD: `let offset = 10; let f = |x| { x + offset }; return f(32);`
  exits 42. The closure reads `offset` from its captured environment.

- Task 04: Add e2e proof program for closures passed as function arguments
  [W18-P02-T04-CLOSURE-AS-ARG-E2E]
  DoD: proof program P1 (closures) from the wave exit criteria passes.

### Phase 03: Trait System Completion [W18-P03-TRAIT-SYSTEM]

The checker resolves trait methods including supertraits, but no e2e test
verifies trait dispatch on concrete types. Several trait features are parsed
but not enforced.

- Task 01: Trait method dispatch on concrete types
  [W18-P03-T01-TRAIT-DISPATCH-E2E]
  Currently: `lookupMethod` in `compiler/check/methods.go:57-104` resolves
  through trait impls. Never tested end-to-end.
  DoD: proof program P2 (trait dispatch) from the wave exit criteria passes.

- Task 02: Enforce trait bounds on generic type parameters
  [W18-P03-T02-TRAIT-BOUNDS]
  Currently: `[T: Display]` is parsed but the bound is ignored during
  checking. Any type can be passed.
  DoD: `fn show[T: Display](x: T)` rejects types without Display impl.

- Task 03: Where clause enforcement
  [W18-P03-T03-WHERE-CLAUSES]
  Currently: `where T: Display` is parsed but never checked.
  DoD: where constraints are validated during type checking. Violations
  produce diagnostics.

- Task 04: Generic impl blocks
  [W18-P03-T04-GENERIC-IMPLS]
  Currently: only concrete impls work. `impl[T] Trait for Container[T]`
  is not handled.
  DoD: a generic impl block is specialized per instantiation. Methods
  on `List[I32]` resolve through `impl[T] List[T]`.

- Task 05: Trait default method implementations
  [W18-P03-T05-DEFAULT-METHODS]
  Currently: traits can declare method signatures but cannot provide
  default bodies.
  DoD: a trait method with a default body is inherited by impls that do
  not override it.

- Task 06: Associated types in traits
  [W18-P03-T06-ASSOCIATED-TYPES]
  Currently: not implemented.
  DoD: `trait Iterator { type Item; fn next(mutref self) -> Option[Self.Item]; }`
  works and the associated type resolves during checking.

- Task 07: Implicit mutref on method receivers
  [W18-P03-T07-IMPLICIT-MUTREF-E2E]
  Currently: language guide contract says `items.push(1)` should work
  without `mutref items`. Not tested.
  DoD: `var v = Vec.new(); v.push(1);` compiles without explicit `mutref`.

### Phase 04: Control Flow Completions [W18-P04-CONTROL-FLOW]

- Task 01: for..in with Iterator protocol
  [W18-P04-T01-FOR-IN-ITERATOR]
  Currently: `for x in coll` is lowered but the binding type is Unknown
  (`lower.go:440`) and the lowerer branches on the iterable directly
  instead of calling `next()` (`lower.go:451`).
  DoD: `for..in` calls `into_iter()` on the collection, then calls
  `next()` in a loop, matching `Some(v)` to continue and `None` to break.
  Proof program P3 passes.

- Task 02: Optional chaining (?.) — full implementation
  [W18-P04-T02-OPTIONAL-CHAINING]
  Currently: parsed and type-checked but returns Unknown type
  (`compiler/check/expr.go:312-316`). Lowering not implemented.
  DoD: `expr?.field` evaluates `expr`, returns None/Err if absent, or
  accesses `field` if present. E2e proof program passes.

- Task 03: Struct and tuple patterns in match
  [W18-P04-T03-STRUCT-TUPLE-PATTERNS]
  Currently: `StructPat` and `TuplePat` are parsed by the parser but have
  no lowering code in the lowerer.
  DoD: `match p { Point { x, y } => x + y }` and
  `match t { (a, b) => a + b }` compile and produce correct results.

### Phase 05: Safety and Destruction [W18-P05-SAFETY-DESTRUCTION]

- Task 01: Drop destructors run at scope exit
  [W18-P05-T01-DROP-E2E]
  Currently: `InstrDrop` emission exists (`codegen/emit.go:252-258`) and
  calls `TypeName_drop(&_lN)` for types with Drop impls. Never tested.
  DoD: e2e proof program P4 verifies a destructor runs. A type with a
  Drop impl sets a flag or modifies observable state that the test checks.

- Task 02: Recursive Drop on compound types
  [W18-P05-T02-RECURSIVE-DROP]
  Currently: only top-level locals get Drop calls. Struct fields with Drop
  implementations are not recursively destroyed.
  DoD: a struct containing a field with Drop has both destructors called
  in the correct order (inner before outer).

- Task 03: Unsafe block enforcement
  [W18-P05-T03-UNSAFE-ENFORCEMENT]
  Currently: `unsafe { }` is parsed as a regular block. No compile error
  when calling extern FFI functions outside unsafe blocks.
  DoD: calling an extern function outside `unsafe { }` produces a
  diagnostic. Calling inside `unsafe { }` compiles successfully.

- Task 04: Recursive type detection
  [W18-P05-T04-RECURSIVE-TYPE-DETECTION]
  Currently: `struct Node { next: Node }` causes infinite-size types.
  No diagnostic.
  DoD: the checker detects directly recursive type definitions and emits
  a diagnostic.

- Task 05: Module visibility enforcement (pub)
  [W18-P05-T05-PUB-VISIBILITY]
  Currently: `pub` is parsed but all symbols are accessible across modules.
  DoD: non-pub items in module A are not accessible from module B.
  Attempting access produces a diagnostic.

### Phase 06: Strings and I/O [W18-P06-STRINGS-AND-IO]

I/O depends on: String, slices, unsafe blocks, and extern FFI. This phase
ensures the full chain works.

- Task 01: String literal to C string emission
  [W18-P06-T01-STRING-LITERALS]
  Currently: string literals are checked as `InternStruct("core", "String")`
  but no C representation exists for string values.
  DoD: `let s = "hello";` produces a C string constant or struct with data
  pointer and length.

- Task 02: String escape sequences
  [W18-P06-T02-STRING-ESCAPES]
  Currently: language guide specifies `\n`, `\r`, `\t`, `\\`, `\"` and
  Unicode escapes. The lexer may not implement the full suite.
  DoD: `"hello\nworld"` produces a string with an embedded newline in
  generated C.

- Task 03: String.len() and String.as_bytes()
  [W18-P06-T03-STRING-METHODS]
  Currently: `stdlib/core/string.fuse` declares these methods but bodies
  are stubs.
  DoD: `"hello".len()` returns 5. `as_bytes()` returns a valid slice.

- Task 04: print() and println() to stdout
  [W18-P06-T04-PRINT-PRINTLN]
  Currently: `stdlib/full/io.fuse` calls `fuse_rt_io_write_stdout` via
  extern FFI. Depends on String.as_bytes() and unsafe blocks.
  DoD: proof program P5 passes — `println("hello")` writes "hello\n" to
  stdout.

- Task 05: File I/O — open, read, write, close
  [W18-P06-T05-FILE-IO]
  Currently: `File` struct in `stdlib/full/io.fuse` with extern FFI.
  DoD: a program opens a temp file, writes data, reads it back, and
  verifies contents match.

### Phase 07: Stdlib Core Tier [W18-P07-STDLIB-CORE]

Every module in `stdlib/core/` must compile through the pipeline and have
its public API exercised by proof programs.

- Task 01: Fix Option[T] impl block to be generic
  [W18-P07-T01-OPTION-IMPL]
  Currently: `impl Option` (not `impl[T] Option[T]`). Methods use `T`
  without it being in scope. `map()` takes `f: Fn` (unparameterized).
  DoD: `Option[I32].unwrap_or(default)` compiles and runs. The impl block
  is `impl[T] Option[T]`.

- Task 02: Fix Result[T, E] impl block to be generic
  [W18-P07-T02-RESULT-IMPL]
  Currently: same problem as Option. `impl Result` is not generic.
  DoD: `Result[I32, Bool].unwrap_or(0)` compiles and runs.

- Task 03: Core traits — implement Equatable and Comparable for primitives
  [W18-P07-T03-CORE-TRAITS-PRIMITIVES]
  Currently: traits declared in `stdlib/core/traits.fuse` but no concrete
  type implements them.
  DoD: `I32` implements `Equatable`. `==` on user types dispatches through
  the trait.

- Task 04: Iterator and IntoIterator traits
  [W18-P07-T04-ITERATOR-TRAITS]
  Currently: `Iterator.next()` returns `Option` (no type parameter).
  `IntoIterator.into_iter()` returns `Self`.
  DoD: `Iterator` has `type Item` associated type. `next()` returns
  `Option[Self.Item]`. Arrays/slices implement `IntoIterator`.

- Task 05: Primitives module — type aliases and primitive methods
  [W18-P07-T05-PRIMITIVES]
  Currently: `stdlib/core/primitives.fuse` declares `type Int = ISize`,
  `type Float = F64` and primitive method stubs. Type aliases are not
  implemented.
  DoD: `Int` resolves to `ISize`. Primitive methods (`abs()`, `min()`,
  `max()`) produce correct results in e2e tests.

- Task 06: Collections — List[T] with push, pop, get
  [W18-P07-T06-COLLECTIONS-LIST]
  Currently: `List[T]` declared with empty method bodies in
  `stdlib/core/collections.fuse`.
  DoD: `List[I32].new()`, `push(42)`, `get(0)` compiles and returns 42.
  Requires: generics, generic impls, runtime memory allocation.

- Task 07: Collections — Map[K, V] and Set[T]
  [W18-P07-T07-COLLECTIONS-MAP-SET]
  Currently: stubs in `stdlib/core/collections.fuse`.
  DoD: `Map[String, I32].insert("key", 42)` and `get("key")` works.
  Requires: Hashable trait, String hashing, generics.

- Task 08: Hash module
  [W18-P07-T08-HASH]
  Currently: `stdlib/core/hash.fuse` exists but not compiled.
  DoD: hash module compiles. Hasher struct produces deterministic hashes.

- Task 09: Clone trait — implement for primitives and common types
  [W18-P07-T09-CLONE]
  Currently: `Clone` trait declared. No implementations.
  DoD: `I32.clone()` works. String.clone() produces a copy.

- Task 10: Display and Debug formatting
  [W18-P07-T10-DISPLAY-DEBUG]
  Currently: traits declared with `Formatter` parameter. No formatting
  infrastructure.
  DoD: `Display` for `I32` produces "42". Basic Formatter exists.

### Phase 08: Stdlib Hosted Tier [W18-P08-STDLIB-HOSTED]

Every module in `stdlib/full/` must compile and have its public API tested.

- Task 01: OS module — argc, argv, env_get, exit
  [W18-P08-T01-OS-MODULE]
  Currently: `stdlib/full/os.fuse` declares FFI wrappers. Not compiled.
  DoD: a program reads argc/argv from the runtime and exits with a code
  derived from the argument count.

- Task 02: Sync module — Mutex, Cond
  [W18-P08-T02-SYNC-MODULE]
  Currently: extern declarations for runtime sync primitives. Not compiled.
  DoD: Mutex lock/unlock compiles and runs. Cond wait/signal compiles.

- Task 03: Thread module — spawn with closures
  [W18-P08-T03-THREAD-MODULE]
  Currently: `thread_spawn()` calls `fuse_rt_thread_spawn`. Depends on
  closure-to-function-pointer conversion.
  DoD: `spawn fn() { ... }` creates a real thread. E2e test verifies
  concurrent execution.

- Task 04: Channel module — send, recv, close with runtime integration
  [W18-P08-T04-CHANNEL-MODULE]
  Currently: `Chan[T]` with stub method bodies. `send()` returns `Ok(())`,
  `recv()` returns `Err("channel empty")`. No runtime queue.
  DoD: proof program P6 (channels) passes. `Chan[I32].send(42)` and
  `recv()` exchange values between threads.

- Task 05: @value struct decorator — auto-derive core traits
  [W18-P08-T05-VALUE-STRUCT]
  Currently: `@value` parsed but no auto-derivation.
  DoD: `@value struct Point { x: I32, y: I32 }` auto-derives Equatable,
  Clone, and Debug.

- Task 06: @rank(N) lock ordering enforcement
  [W18-P08-T06-RANK-ENFORCEMENT]
  Currently: parsed but not enforced.
  DoD: acquiring locks out of rank order produces a compile-time diagnostic.

### Phase 09: Generic Type Inference Improvements [W18-P09-GENERIC-INFERENCE]

- Task 01: Infer full type args from usage context
  [W18-P09-T01-CONTEXTUAL-INFERENCE]
  Currently: `let r = Ok(42)` produces `Result[I32, Unknown]` instead of
  `Result[I32, Bool]` when used as `try_get[I32, Bool](r)`.
  DoD: the checker propagates expected types from call-site parameter types
  back to variable definitions. `let r = Ok(42); f[I32, Bool](r)` types
  `r` as `Result[I32, Bool]`.

- Task 02: Infer generic args from return type context
  [W18-P09-T02-RETURN-TYPE-INFERENCE]
  Currently: explicit type args required on calls like `identity[I32](42)`.
  DoD: `fn main() -> I32 { return identity(42); }` infers T=I32 from
  the return position without explicit `[I32]`.

- Task 03: Zero-argument generic calls with explicit type args
  [W18-P09-T03-ZERO-ARG-GENERICS]
  Currently: `sizeOf[I32]()` pattern not tested.
  DoD: generic functions with no value arguments specialize correctly
  when explicit type args are provided.

### Phase 10: Regression Closure and Proof Program Audit [W18-P10-REGRESSION-CLOSURE]

- Task 01: Verify every L016 item (1–40) has a proof program or is descoped
  [W18-P10-T01-L016-AUDIT]
  DoD: every item in learning-log L016 is either covered by an e2e test
  or explicitly removed from the language guide with a rationale.

- Task 02: Compile stdlib/core/ through the full pipeline
  [W18-P10-T02-STDLIB-CORE-COMPILES]
  DoD: `fuse check stdlib/core/` succeeds with no type errors. Every
  public method body is checked.

- Task 03: Compile stdlib/full/ through the full pipeline
  [W18-P10-T03-STDLIB-FULL-COMPILES]
  DoD: `fuse check stdlib/full/` succeeds with no type errors.

- Task 04: Stage 2 re-verification with full feature set
  [W18-P10-T04-STAGE2-REVERIFY]
  DoD: stage1 compiles stage2, stage2 compiles itself, reproducibility
  check passes. The bootstrap gate holds with the expanded feature set.

- Task 05: Update learning log with any new lessons
  [W18-P10-T05-UPDATE-LEARNING-LOG]
  DoD: learning-log entries exist for any design gaps or architectural
  lessons discovered during this wave.

- Task 06: Verify all proof programs pass in CI
  [W18-P10-T06-CI-PROOF-PROGRAMS]
  DoD: the e2e test suite including all proof programs from Waves 17 and
  18 passes in the CI matrix (Linux, macOS, Windows).

## Wave 19: Retirement of Go and C from the Compiler Path

Goal: complete the transition from bootstrap implementation languages to a Fuse
compiler implemented and built by Fuse.

Entry criterion: Wave 18 done.

Exit criteria:

- Fuse owns the compiler implementation path
- Go is no longer required to build the compiler
- C is no longer required as a backend or runtime implementation dependency in
  the compiler path

### Phase 01: Retire Go [W19-P01-RETIRE-GO]

- Task 01: Freeze Stage 1 as archival bootstrap tool [W19-P01-T01-FREEZE-STAGE1]
  DoD: Stage 1 is no longer required for ordinary compiler development.
- Task 02: Remove Go from active compiler build workflow
  [W19-P01-T02-REMOVE-GO-FROM-WORKFLOW]
  DoD: supported build path no longer invokes Go.

### Phase 02: Retire C [W19-P02-RETIRE-C]

- Task 01: Replace C runtime dependencies as required
  [W19-P02-T01-REPLACE-C-RUNTIME]
  DoD: compiler implementation path no longer requires C runtime code.
- Task 02: Remove C from compiler bootstrap assumptions
  [W19-P02-T02-REMOVE-C-FROM-BOOTSTRAP-ASSUMPTIONS]
  DoD: compiler implementation is Fuse-only.

## Wave 20: Targets and Ecosystem Growth

Goal: resume broader target and library work on top of the self-hosted native
compiler.

Entry criterion: Wave 19 done.

Exit criteria:

- target expansion and library growth occur without reintroducing bootstrap debt

### Phase 01: Additional Targets [W20-P01-ADDITIONAL-TARGETS]

- Task 01: Add target descriptions [W20-P01-T01-TARGET-DESCRIPTIONS]
  DoD: each supported target has a documented ABI and validation path.
- Task 02: Add target CI [W20-P01-T02-TARGET-CI]
  DoD: target regressions are visible immediately.

### Phase 02: Extended Libraries [W20-P02-EXTENDED-LIBRARIES]

- Task 01: Implement ext stdlib modules [W20-P02-T01-EXT-STDLIB]
  DoD: optional libraries build on the stable core and hosted tiers.
- Task 02: Publish ecosystem guidance [W20-P02-T02-ECOSYSTEM-GUIDANCE]
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
- Every wave that introduces a user-visible feature must include at least one
  end-to-end proof program: a Fuse source file that compiles, links, runs, and
  produces a verified output. The proof program must fail if the feature is
  stubbed (Rule 6.8).
- Exit criteria must include behavioral requirements ("this program produces
  exit code N"), not only structural ones ("HIR nodes carry metadata").
  Structural criteria are necessary but never sufficient alone (Rule 6.10).
- Every task must name what it replaces: "currently X is stubbed at file:line,
  producing behavior Y." This forces an audit of the current state before
  claiming work is complete.
- Stubs must emit compiler diagnostics, not silent defaults. A feature that
  parses and type-checks but is not lowered must produce an error, not a
  silently wrong program (Rule 6.9).