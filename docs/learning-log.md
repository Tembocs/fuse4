# Fuse Learning Log

> Status: normative for the next production attempt of Fuse (`fuse4`).
>
> This file is the append-only project learning log. It records lessons that the
> team wants the future repository to preserve from the first day of work.

## Entry format

Every entry must use the following structure.

### LNNN — Title

Date: YYYY-MM-DD
Discovered during: Wave / Phase / Task

**Reproducer**:
Minimal case that exposes the problem.

**What was tried first**:
The first failed approach and why it failed.

**Root cause**:
What was actually wrong.

**Spec gap**:
Which part of the language guide was silent, ambiguous, or incomplete.

**Plan gap**:
Which part of the implementation plan failed to schedule or constrain the work.

**Fix**:
What changed.

**Cascading effects**:
What other bugs or design consequences the fix exposed.

**Architectural lesson**:
What invariant or design principle should be carried forward.

**Verification**:
The commands, tests, or fixtures that proved the fix.

## Entries

### L000 — Learning log format

Date: 2026-04-14
Discovered during: Wave 00 / Phase 01 / Task 02

**Reproducer**:
Not applicable. This entry establishes the required log format.

**What was tried first**:
Previous attempts recorded lessons informally, but the format did not reliably
capture the specification and planning consequences of each bug.

**Root cause**:
A bug log that records chronology without forcing a spec gap and a plan gap does
not reliably improve the next iteration.

**Spec gap**:
The earlier process did not require each meaningful bug to feed back into the
language guide.

**Plan gap**:
The earlier process did not require each meaningful bug to map back into the
implementation plan.

**Fix**:
Adopt the structured entry format defined above from the beginning of the
project.

**Cascading effects**:
Future bugs become easier to classify into language, planning, implementation,
or tooling failures.

**Architectural lesson**:
The learning log is useful only if it tightens the guide and plan, not if it is
used as a loose diary.

**Verification**:
Every future learning-log entry must conform to this format.

### L001 — Lexical ambiguities must become explicit contracts

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from Fuse 3

**Reproducer**:
Inputs such as `r#abc` and `parse(x)?.field` produced incorrect lexical or parse
behavior when the scanner and parser relied on intuitive rather than explicit
rules.

**What was tried first**:
The earlier implementation assumed that raw strings and optional chaining would
fall out naturally from token longest-match and ordinary postfix parsing.

**Root cause**:
The language specification described the features for users, but did not define
the precise recognition and parse contracts an implementation needed.

**Spec gap**:
The language guide was missing explicit implementation contracts for raw-string
recognition, `?.` tokenization, and struct-literal disambiguation.

**Plan gap**:
The lexer and parser waves did not schedule ambiguity-closure tasks explicitly.

**Fix**:
Carry these rules into the new language guide as mandatory implementation
contracts and schedule ambiguity-specific regression work in the early waves.

**Cascading effects**:
The parser and lexer test corpus must include ambiguity-focused golden cases, not
just representative examples.

**Architectural lesson**:
Surface syntax is insufficient when ambiguity exists. The specification must say
how the compiler chooses.

**Verification**:
The new language guide includes explicit contracts for raw strings, `?.`, and
struct-literal disambiguation, and the new implementation plan schedules those
tasks in Waves 01 and 02.

### L002 — Stdlib body checking is mandatory, not optional

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from Fuse 3

**Reproducer**:
Skipping stdlib body checking while still lowering and codegening stdlib modules
caused large numbers of `Unknown` types to propagate into generated C as `int`.

**What was tried first**:
The earlier compiler treated stdlib signatures as enough to move forward while
deferring body checking for speed and convenience.

**Root cause**:
Frontend completeness was broken across pass boundaries: later passes consumed
stdlib HIR whose expressions had never been semantically completed.

**Spec gap**:
The language guide and rules did not state strongly enough that stdlib modules
must be checked like user modules if they participate in lowering and codegen.

**Plan gap**:
The type-checking wave did not make stdlib body checking an explicit exit
criterion from the beginning.

**Fix**:
State the rule in the language guide and rules document, and create a dedicated
phase for stdlib body checking in the implementation plan.

**Cascading effects**:
Once stdlib bodies are checked, many latent semantic gaps surface earlier and in
the correct subsystem.

**Architectural lesson**:
If a module reaches lowering, it must already be semantically complete.

**Verification**:
The new language guide, rules, and implementation plan all make stdlib body
checking mandatory.

### L003 — Monomorphization must reject partial specializations

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from Fuse 3

**Reproducer**:
Generic functions and impl methods could produce plausible-looking specialized
names even when some required type parameters remained unresolved.

**What was tried first**:
The earlier implementation mainly guarded against obviously unknown type
arguments rather than checking completeness of the whole substitution set.

**Root cause**:
Monomorphization validity was defined too loosely. The system treated “some
inference succeeded” as good enough.

**Spec gap**:
The language guide lacked an explicit rule that a valid specialization requires
all function and impl type parameters to be substituted concretely.

**Plan gap**:
Generic specialization validity was not scheduled as its own phase with its own
regression closure.

**Fix**:
Define specialization completeness explicitly in the guide and give
monomorphization its own wave and phases in the plan.

**Cascading effects**:
Zero-argument constructor-style generics and explicit type-argument calls must be
handled deliberately.

**Architectural lesson**:
Partial specialization is worse than no specialization because it poisons later
passes with believable garbage.

**Verification**:
The new language guide and implementation plan both define completeness and make
recursive concreteness checks mandatory.

### L004 — Pointer categories are a backend architecture rule

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from Fuse 3

**Reproducer**:
The same backend logic handled borrow-derived pointers and `Ptr[T]` values as if
they had identical semantics, causing miscompilation at assignments, returns,
and field accesses.

**What was tried first**:
The earlier backend relied on local heuristics around whether a value's C type
was pointer-shaped.

**Root cause**:
The backend lacked a formal distinction between pointer representations arising
from borrow semantics and pointer values that are part of the language.

**Spec gap**:
The language guide did not define the two pointer categories explicitly.

**Plan gap**:
The codegen wave did not schedule pointer-category handling as a first-class
contract.

**Fix**:
Document the two-pointer-category model in the guide and give it an explicit
phase in the backend wave.

**Cascading effects**:
Call-site adaptation and field-access lowering both depend on this distinction.

**Architectural lesson**:
Backend representation rules are architecture, not cleanup details.

**Verification**:
The new language guide and implementation plan both include a dedicated pointer
category contract.

### L005 — Unit erasure must be total and global

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from Fuse 3

**Reproducer**:
Partially erased unit payloads and parameters produced ghost data paths,
nonexistent reads, and invalid function-pointer shapes in generated C.

**What was tried first**:
The earlier implementation applied unit erasure opportunistically in some codegen
sites but not others.

**Root cause**:
Erasure was treated as a local optimization instead of a global ABI decision.

**Spec gap**:
The language guide did not state that once unit is erased in one location, it is
erased everywhere that participates in the same concrete ABI.

**Plan gap**:
The lowering and backend waves did not isolate unit erasure as an explicit task.

**Fix**:
State total unit erasure as a hard implementation contract and schedule it as its
own backend phase.

**Cascading effects**:
Constructors, patterns, function pointers, and aggregate layout must all agree.

**Architectural lesson**:
There is no such thing as partially erased unit.

**Verification**:
The new language guide encodes total unit erasure and the new implementation plan
gives it dedicated backend tasks.

### L006 — Divergence must be structural, not simulated

Date: 2026-04-14
Discovered during: Pre-bootstrap carryover from Fuse 3

**Reproducer**:
Lowering and codegen that simulated post-divergence values produced references to
undeclared temporaries after calls to panic-like functions.

**What was tried first**:
The earlier backend attempted to satisfy type expectations by inventing fallback
values after control flow had already diverged.

**Root cause**:
Divergence was treated as a typing inconvenience instead of as a fundamental
control-flow property.

**Spec gap**:
The language guide did not define divergence as a structural MIR and backend
property strongly enough.

**Plan gap**:
The lowering and backend waves did not schedule divergence closure as its own
explicit responsibility.

**Fix**:
Document structural divergence in the guide and plan, and make it part of both
lowering and backend exit criteria.

**Cascading effects**:
Join blocks, aggregate fallbacks, and destruction paths all depend on accurate
divergence handling.

**Architectural lesson**:
If control flow does not continue, the compiler must stop pretending that it
does.

**Verification**:
The new language guide and implementation plan both treat divergence as a
structural contract.

### L007 — Pattern matching must dispatch on discriminants, not fall through

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
A `match` expression with multiple arms compiles, but the generated code jumps
unconditionally to the first arm's block. All subsequent arms are dead code.

**What was tried first**:
The lowerer emitted `TermGoto(armBlock)` for each arm without evaluating the
pattern. This made match expressions parse, type-check, and "work" in tests
that only had one arm.

**Root cause**:
Match lowering was left as a stub. HIR stores patterns as text strings
(`PatternDesc string`) instead of structured pattern nodes, making real dispatch
impossible at the MIR level.

**Spec gap**:
The language guide defines pattern matching semantics, but the HIR and MIR
specifications did not mandate structured pattern representation.

**Plan gap**:
No wave or phase owns pattern lowering as an explicit task. Wave 07 (HIR→MIR)
mentions control flow but does not list match dispatch. Wave 05 (type checking)
mentions match but does not require exhaustiveness.

**Fix**:
1. Add structured pattern nodes to HIR (LiteralPat, BindPat, ConstructorPat,
   WildcardPat).
2. Lower match expressions to cascading branch chains in MIR using enum
   discriminant comparison.
3. Emit correct `TermBranch` / `TermSwitch` sequences in codegen.

**Cascading effects**:
Enum destructuring, exhaustiveness checking, and guard expressions all depend on
real pattern dispatch.

**Architectural lesson**:
A stub that compiles without error is more dangerous than a stub that crashes.
Stubs must be tracked to completion or produce diagnostics.

**Verification**:
Match expressions with multiple arms produce distinct codegen paths, tested via
unit tests on the lowerer and codegen.

### L008 — Monomorphization cannot be deferred past codegen

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
A generic function `fn id[T](x: T) -> T { x }` type-checks, but no concrete
specialization is collected or emitted. Any program using `Option[I32]`,
`Result[T, E]`, or other generic types cannot produce working binaries.

**What was tried first**:
The bootstrap path avoided generics entirely. The Stage 2 compiler and its tests
use only concrete types, so the self-hosting gate (Wave 15) passed without
monomorphization.

**Root cause**:
The `compiler/monomorph/` package was created as a placeholder but never
implemented. No wave in the implementation plan owns generic specialization as a
task with entry/exit criteria.

**Spec gap**:
The language guide defines generics and monomorphization, but the implementation
plan does not schedule the work.

**Plan gap**:
Wave 05 mentions generic inference. Wave 07 mentions lowering. Neither owns the
actual collection of concrete instantiations or the expansion of generic
function bodies with concrete types. The monomorph package is referenced in the
repository layout but has no corresponding wave tasks.

**Fix**:
1. Implement `monomorph.Collect()` to scan all call sites and collect concrete
   type argument sets.
2. Implement `monomorph.Specialize()` to produce concrete MIR functions from
   generic HIR templates.
3. Integrate into the driver pipeline between type checking and MIR lowering.

**Cascading effects**:
All generic stdlib types (Option, Result, List, Map, Set) and user-defined
generic functions require this to produce working code.

**Architectural lesson**:
A placeholder package with a doc.go is not a substitute for a scheduled,
tested implementation. If a feature has no wave task, it will not be built.

**Verification**:
Generic functions produce specialized MIR and correct C output, tested via
unit tests on monomorph and codegen.

### L009 — Error propagation operator must lower to control flow

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
The `?` operator on a `Result[T, E]` expression compiles, but the checker
returns `Unknown` type and the lowerer simply unwraps the inner expression
without any error checking or early return.

**What was tried first**:
The checker and lowerer treated `?` as a pass-through: `checkQuestion()` returns
`Unknown`, and `lowerExpr(QuestionExpr)` returns `lowerExpr(n.Expr)`. This
allowed the pipeline to proceed without crashing.

**Root cause**:
The `?` operator requires knowledge of the Result/Option type structure to
extract the success value and propagate the error. Without monomorphization
and concrete enum layout, this was deferred — but no task tracked its
completion.

**Spec gap**:
The language guide defines `?` semantics, but the HIR and lowering contracts
do not specify how `?` maps to branching control flow.

**Plan gap**:
No wave or phase owns the `?` operator implementation. Wave 05 type-checks it
as Unknown. Wave 07 lowers it as a no-op.

**Fix**:
1. Checker: extract the inner `T` from `Result[T, E]` or `Option[T]` and
   return it as the expression type.
2. Lowerer: emit a branch that checks for Err/None and early-returns if so,
   otherwise continues with the unwrapped value.
3. Codegen: standard branch emission handles this naturally.

**Cascading effects**:
Depends on enum discriminant access (pattern matching) and knowledge of
Result/Option layout (monomorphization or special-casing).

**Architectural lesson**:
Operators that affect control flow cannot be stubbed as expression-level
pass-throughs. They must produce branches or they silently corrupt behavior.

**Verification**:
`?` on Result/Option produces early-return branches in MIR, tested via
unit tests on check and lower.

### L010 — Drop codegen must emit actual destructor calls

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
The liveness pass correctly computes `DestroyEnd` flags, and the lowerer emits
`EmitDrop` instructions, but the C11 backend emits only `/* drop _lN */`
comments. No actual cleanup code runs at runtime.

**What was tried first**:
The codegen emitted comments as placeholders, intending to revisit drop emission
later. Because no test actually ran the generated C with destructor-dependent
resources, the gap was invisible.

**Root cause**:
Drop emission requires knowing whether a type has a `Drop` trait implementation.
Without that metadata flow from check → codegen, the backend cannot emit the
correct destructor call.

**Spec gap**:
The language guide defines deterministic destruction, but the backend contracts
do not specify how `InstrDrop` maps to C code.

**Plan gap**:
Wave 06 (ownership/liveness) schedules drop intent insertion, but no wave
schedules the codegen side — the actual C emission of destructor calls.

**Fix**:
1. Flow Drop-trait information from the checker into the type table or a
   side table accessible during codegen.
2. Codegen: emit `TypeName_drop(&_lN);` for types with Drop impls;
   no-op for types without.
3. Test with types that have explicit Drop implementations.

**Cascading effects**:
Resource management (file handles, locks, allocations) depends on actual
destructor calls, not comments.

**Architectural lesson**:
A comment is not a drop. If codegen emits a comment where it should emit code,
the feature does not exist.

**Verification**:
InstrDrop for types with Drop impls emits function calls in generated C, tested
via codegen unit tests.

### L011 — Closures must capture environments, not erase to unit

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
A closure expression `|x| { x + 1 }` type-checks and produces a valid function
type, but the lowerer returns `constUnit()` — the closure body is never lowered
to MIR and no environment capture occurs.

**What was tried first**:
The lowerer treated closures as "function references (simplified)" and returned
unit. Liveness analysis also skips closure bodies entirely.

**Root cause**:
Closures require environment capture analysis (which outer variables are
referenced), allocation of a closure struct, and emission of a lifted function.
This is a non-trivial transformation that was deferred without a plan task.

**Spec gap**:
The language guide describes closures but does not specify the lowering
representation (lifted function + environment struct).

**Plan gap**:
No wave owns closure lowering. Wave 07 (HIR→MIR) does not mention closures.

**Fix**:
1. Implement capture analysis: scan closure bodies for references to outer
   variables.
2. Generate an environment struct type with captured variables.
3. Lift the closure body to a standalone MIR function that takes the
   environment as a parameter.
4. At the closure expression site, emit struct init for the environment and
   a function pointer pair.

**Cascading effects**:
Iterators, callbacks, and higher-order functions all depend on closures.

**Architectural lesson**:
A feature that type-checks but does not lower is a silent miscompilation, not a
deferred feature.

**Verification**:
Closures produce lifted functions and environment structs in MIR and C output,
tested via unit tests on lower and codegen.

### L012 — Channels must exist as types before concurrency can work

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit

**Reproducer**:
The stdlib defines `chan.fuse` with channel operations, but no channel type
exists in the type table or compiler. `spawn` expressions lower to plain
function calls with no threading semantics.

**What was tried first**:
The lowerer treats `SpawnExpr` as `EmitCall(dest, arg, nil, Unit, false)` — a
synchronous function call. No thread creation occurs.

**Root cause**:
Channel types and spawn semantics require runtime integration (thread creation,
queue management) that was deferred. The stdlib `chan.fuse` file defines the
API surface but the compiler has no knowledge of channel types.

**Spec gap**:
The language guide describes channels and spawn as language primitives, but the
type table and backend contracts do not include them.

**Plan gap**:
Wave 08 (runtime) implements thread and sync primitives in C, but no wave
schedules the compiler-side integration: channel type interning, spawn lowering
to `fuse_rt_thread_spawn`, or channel operation lowering to runtime calls.

**Fix**:
1. Add channel type kind to the type table.
2. Lower `spawn expr` to a runtime call: `fuse_rt_thread_spawn(fn, arg)`.
3. Lower channel operations (send, recv, close) to corresponding runtime calls.
4. Type-check channel expressions with proper generic element types.

**Cascading effects**:
All concurrency features in the language depend on channels and proper spawn.

**Architectural lesson**:
A runtime library without compiler integration is dead code. Both sides must be
scheduled together.

**Verification**:
Spawn emits `fuse_rt_thread_spawn` calls and channel operations emit runtime
calls, tested via codegen unit tests.

### L013 — Self-verifying plans are not verification

Date: 2026-04-14
Discovered during: Pre-Wave-17 audit, after implementing L007–L012 fixes

**Reproducer**:
Six critical compiler features (pattern matching, monomorphization, error
propagation, drop codegen, closures, channels) were stubbed or missing despite
the implementation plan showing Waves 00–16 as complete. Every wave's exit
criteria were satisfied. Every test passed. The compiler reached the self-hosting
gate at Wave 15 and the native backend transition at Wave 16 with features that
had never produced a working program.

After fixing all six features, the same pattern repeated: the AST-to-HIR bridge
was built with `Unknown` types for all expressions, which made e2e tests pass
for simple programs but left the newly implemented features (generics, pattern
matching, closures, `?` operator) unreachable from any end-to-end test.

**What was tried first**:
Each wave was implemented to satisfy its stated exit criteria. The plan, the
implementation, the tests, and the verification were all produced by the same
agent in the same session. Unit tests were written for the features, and they
passed. The wave was declared complete.

**Root cause**:
The plan, the implementation, and the tests formed a closed loop with no
external forcing function. The agent wrote exit criteria it could satisfy, built
implementations that satisfied those criteria, and wrote tests that validated the
implementations. At no point did an independent check ask: "compile a real
program that uses this feature and run it."

Specifically:
- Exit criteria were phrased as structural properties ("MIR blocks terminate
  structurally") rather than behavioral requirements ("a program using match
  with three enum variants produces the correct output").
- The self-hosting gate (Wave 15) passed because the Stage 2 compiler source
  does not use generics, closures, pattern matching with payloads, or the `?`
  operator. The gate tested whether Fuse can compile *itself*, not whether Fuse
  can compile *programs*.
- Unit tests validated individual components in isolation. No test compiled a
  Fuse program through the complete pipeline and executed the resulting binary
  to check its output.
- The AST-to-HIR bridge defaulted all types to `Unknown`, which mapped to C
  `int`, which compiled and ran correctly for integer-only programs. The bridge
  was "working" in the same way the stubs were "working" — it satisfied the
  tests that existed without satisfying the purpose it was built for.

**Spec gap**:
The implementation plan does not require behavioral end-to-end tests as exit
criteria for any wave. Structural correctness ("HIR nodes carry metadata",
"MIR is property-tested") is necessary but not sufficient.

**Plan gap**:
No wave requires a program that exercises the wave's feature to compile, link,
run, and produce verified output. The plan's verification model is entirely
structural and internal. There is no external validation step.

**Fix**:
1. Every wave that introduces a user-visible feature must include at least one
   end-to-end test that compiles a Fuse program using that feature, runs the
   binary, and checks the output.
2. Exit criteria must include behavioral requirements, not only structural ones.
   "Pattern matching works" means "a program with a match expression on an enum
   with three variants returns the correct arm's value when executed."
3. The AST-to-HIR bridge must propagate the checker's resolved types so that
   features beyond integer arithmetic are reachable from e2e tests.
4. When an agent produces a plan and then implements it, the verification step
   must be adversarial: "write a program that would fail if this feature were
   stubbed, then run it."

**Cascading effects**:
Every future wave must be accompanied by e2e test programs that exercise the
feature. The e2e test suite becomes a release gate alongside unit tests. The
AST-to-HIR bridge must be completed with real type flow before any feature
can be considered truly implemented.

**Architectural lesson**:
A plan that an agent writes and then satisfies is not a plan — it is a
self-fulfilling prophecy. Verification must be independent of the implementer.
When the same agent writes the criteria, the implementation, and the tests, the
only reliable check is a concrete program that runs and produces the right
answer. Structural tests prove the code compiles. Behavioral tests prove the
code works.

**Verification**:
This entry is verified by the existence of L007–L012 (six features that passed
all structural tests while being stubbed) and by the current state of the e2e
suite (21 tests that compile and run programs, but none that exercise generics,
pattern matching on enums, closures, error propagation, or drop behavior).

### L014 — Document requirements for preventing self-verifying plans

Date: 2026-04-14
Discovered during: Post-audit review of foundational document effectiveness

**Reproducer**:
L013 identified that the plan, implementation, and tests formed a closed loop.
This entry records the concrete requirements that each foundational document
must satisfy to prevent that failure pattern from recurring.

**What was tried first**:
The five foundational documents were written with structural completeness as
the standard. The language guide described features. The plan scheduled tasks.
The rules defined invariants. The layout defined placement. The learning log
recorded bugs. None of them required a running program as evidence that a
feature works.

**Root cause**:
The documents governed *how* to build the compiler but not *how to prove* it
works. Structural properties (HIR carries metadata, MIR terminates, codegen
emits typed initializers) are necessary but are not proof of correctness. A
stub that returns unit satisfies every structural property while producing
wrong behavior. Only a program that compiles, runs, and produces the expected
output proves a feature works.

**Spec gap**:
The language guide describes features without requiring compilable proof
programs. A feature section that says "Fuse has pattern matching" without a
runnable example that the compiler must handle is an untestable claim.

**Plan gap**:
The implementation plan defines exit criteria as structural properties, not
behavioral outcomes. No wave requires a proof program.

**Fix**:
The following requirements are normative for all five foundational documents
going forward.

**implementation-plan.md** must satisfy:
1. Every wave that introduces a user-visible feature must include a "proof
   program" — a concrete Fuse program that would fail to compile or produce
   wrong output if the feature were stubbed. The proof program is part of the
   wave's exit criteria.
2. Exit criteria must be behavioral: "this program compiles, runs, and
   returns exit code N" or "this program prints X to stdout." Structural
   criteria ("HIR nodes carry metadata") are permitted only alongside
   behavioral criteria, never alone.
3. Every task must name what it replaces: "currently X is stubbed at file:line,
   producing behavior Y." This forces an audit of current state before claiming
   work is complete.
4. Cross-wave dependencies must be explicit: "Wave 07 lowering depends on
   Wave 05 producing resolved types for generic instantiations. The proof
   program for Wave 07 must use a generic type to exercise this dependency."

**language-guide.md** must satisfy:
1. Every feature section must end with a compilable Fuse example and its
   expected output (exit code or stdout). These examples are the source of
   truth for the e2e test suite.
2. Every operator and control structure must have a behavioral contract:
   what the generated code must do, not just what the syntax looks like.
   "`?` propagates errors" is insufficient. "`?` on `Result[T, E]` evaluates
   the expression; if `Ok(v)`, evaluates to `v`; if `Err(e)`, the enclosing
   function returns `Err(e)` immediately; the generated code must contain a
   branch" is a testable contract.
3. Features must be marked as "implemented" or "specified but not yet
   implemented." Silence means "this works" — and that claim must be
   testable by running the section's example program.

**rules.md** must satisfy:
1. Add rule: "No feature is complete until a program using it compiles, links,
   runs, and produces the correct output." Unit tests are necessary but not
   sufficient.
2. Add rule: "Stubs must emit diagnostics, not silent defaults." A closure
   lowering that returns unit must emit `error: closures are not yet
   implemented`. A `?` operator that returns Unknown must emit `error: error
   propagation is not yet implemented`. A stub that compiles silently is
   indistinguishable from a working implementation.
3. Add rule: "Exit criteria written by the implementer must be validated
   against the proof program before the wave is marked complete."

**repository-layout.md** must satisfy:
1. The `tests/e2e/` directory must be populated from Wave 01 onward. Every
   compiler package that affects program behavior must have a corresponding
   e2e fixture program.
2. Proof programs are checked-in `.fuse` source files with expected outputs,
   not programmatically generated Go test scaffolding. Anyone must be able to
   read `tests/e2e/pattern_match.fuse` and understand what the compiler is
   supposed to do.

**learning-log.md** (this document) must satisfy:
1. Before closing any wave, ask: "Could this wave's implementation produce a
   learning-log entry if audited independently?" If yes, the wave is not
   ready to close.

**Cascading effects**:
All existing waves (00–16) must be retroactively checked against these
requirements. The e2e test suite must be expanded with proof programs for every
implemented feature. The AST-to-HIR bridge must propagate checker types so that
proof programs for generics, pattern matching, closures, and error propagation
can actually execute.

**Architectural lesson**:
Documents that govern construction without governing proof are aspirational,
not normative. A running program is the only proof that a feature exists. A
passing unit test proves the component works in isolation. A passing e2e test
proves the compiler works for the user.

**Verification**:
This entry is verified when: (a) rules.md, implementation-plan.md,
language-guide.md, and repository-layout.md are updated to include the
requirements above; (b) every implemented feature has a proof program in
`tests/e2e/`; and (c) the e2e suite fails if any feature is reverted to a
stub.

### L015 — Generics require a dedicated wave with proof programs at every phase

Date: 2026-04-15
Discovered during: Pre-Wave-17 planning, after L007–L014 audit

**Reproducer**:
The monomorphization package (`compiler/monomorph/`) was implemented with
`Record`, `Substitute`, and `IsGeneric` methods. Unit tests pass. But no generic
Fuse program has ever compiled to a working binary. The package is not integrated
into the driver pipeline, no code scans call sites for generic instantiations,
no code duplicates function bodies with substituted types, and no code rewrites
call sites to reference specialized names.

The implementation plan placed monomorphization in Wave 05 Phase 06 (four tasks)
but the tasks described the monomorph package internals, not the pipeline
integration or end-to-end behavior. A program like
`fn identity[T](x: T) -> T { return x; } fn main() -> I32 { return identity[I32](42); }`
has never been tested.

**What was tried first**:
Monomorphization was added as a phase within the type-checking wave (Wave 05)
because it is conceptually related to type resolution. The four tasks described
collecting instantiations, validating completeness, specializing functions, and
integrating into the pipeline. Each task had a DoD. The monomorph package was
implemented and unit-tested.

**Root cause**:
Generics touch every stage of the pipeline: parsing (generic params), resolution
(type param scoping), checking (type arg inference), monomorphization
(collection and substitution), AST-to-HIR bridge (body duplication), lowering
(concrete types in MIR), and codegen (specialized function names and type
layouts). Cramming this into a single phase of another wave hid the cross-cutting
dependencies. Each component was built in isolation and none were connected.

The specific gaps:
1. The checker does not register generic type parameters as in-scope types
   during body checking.
2. The checker does not resolve explicit type arguments at call sites.
3. No code scans the checked AST to collect concrete instantiations.
4. No code duplicates generic function bodies with concrete type substitution.
5. No code generates specialized function names.
6. No code rewrites call sites to reference specialized names.
7. The driver does not run monomorphization between checking and HIR building.
8. Generic functions with unresolved type parameters are not skipped in codegen.
9. Generic enum types (Option, Result) have no concrete field layout in codegen.
10. The `?` operator depends on specialized Result/Option layout that does not
    exist.

**Spec gap**:
The language guide describes generics and monomorphization but does not specify
the concrete compilation model: what a specialized function looks like in the
generated code, how call sites reference it, or how generic type layouts map to
C struct definitions.

**Plan gap**:
The implementation plan placed monomorphization as a phase within Wave 05 with
four tasks. The tasks were structural ("collect instantiations", "validate
completeness") rather than behavioral ("this program compiles and runs"). No
proof program was required. The cross-cutting nature of generics — touching
parser, checker, monomorphizer, bridge, lowerer, and codegen — was not reflected
in the task structure.

**Fix**:
Create a dedicated Wave 17 (Generics End-to-End) with 10 phases and proof
programs at every integration point:
- Phase 01: Generic parameter scoping in the checker
- Phase 02: Instantiation collection in the driver
- Phase 03: Generic function body specialization
- Phase 04: Driver pipeline integration
- Phase 05: Proof program P1 (basic generic function)
- Phase 06: Multiple instantiations
- Phase 07: Generic types (Option, Result) with concrete layouts
- Phase 08: Enum construction and destructuring with generics
- Phase 09: Error propagation with generics
- Phase 10: Regression closure

Each phase has a behavioral exit criterion. Phases 05, 06, 08, and 09 require
e2e proof programs that compile, run, and produce the correct exit code.

**Cascading effects**:
The existing Wave 17 (Retirement of Go and C) and Wave 18 (Targets and
Ecosystem) are renumbered to Wave 18 and Wave 19. Generics must work before
the bootstrap path is retired, because a self-hosted compiler that cannot
compile generic programs is not a complete compiler.

**Architectural lesson**:
Cross-cutting features cannot be implemented as a phase within a single wave.
When a feature touches every stage of the pipeline, it needs its own wave with
its own entry criteria, exit criteria, and proof programs. The granularity of
the wave must match the granularity of the integration risk.

**Verification**:
This entry is verified when Wave 17 Phase 05 Task 01 passes: the proof program
`fn identity[T](x: T) -> T { return x; } fn main() -> I32 { return identity[I32](42); }`
compiles, runs, and exits with code 42.

### L016 — Pre-Wave-18 completeness audit: language features not yet proven

Date: 2026-04-15
Discovered during: Post-Wave-17 audit, before Wave 18 (Retirement of Go and C)

**Reproducer**:
Wave 17 (Generics End-to-End) is complete with 6 e2e proof programs. But a
self-hosted compiler that cannot compile programs using closures, threads,
strings, I/O, or the standard library is not ready for Wave 18. This entry
catalogs every feature gap that must be closed before Go and C can be retired.

**What was tried first**:
After completing Wave 17, an audit was performed against the language guide,
stdlib source, and compiler pipeline. Features were classified as: proven
(e2e test exists), implemented but unproven (code exists, no e2e test),
partially implemented (incomplete or stub), or specified but not implemented.

**Root cause**:
The implementation plan treated Waves 00–16 as done, and Wave 17 added
generics. But the L013 audit revealed 6 silently stubbed features. After
fixing those and completing generics, the remaining features were never
re-audited against the full language guide and stdlib. The compiler can compile
programs that use arithmetic, control flow, functions, generics, enums, pattern
matching, and error propagation. It cannot compile programs that use closures,
strings, I/O, iteration, channels, trait-driven dispatch, or most stdlib APIs.

**Spec gap**:
The language guide specifies closures, concurrency, iterators, trait methods,
string operations, and I/O as core features. None of these have e2e proof
programs. The stdlib source files declare methods that would fail at type-check
or codegen if actually compiled through the pipeline.

**Plan gap**:
The implementation plan does not have a wave between Wave 17 (Generics) and
Wave 18 (Retirement) that closes the remaining feature gaps. Wave 18 assumes
the language is complete; it is not.

**Fix**:
A new wave (or extension of existing waves) must close every gap below before
Wave 18 begins. Each category lists features from most to least critical.

## Category 1: Implemented but no e2e proof program

These features have code in the parser, checker, lowerer, and codegen, but no
program has ever been compiled through the full pipeline, linked, executed, and
verified. Per Rule 6.8, they are not complete until proven.

1. **Closures** — `|x| { x + 1 }`. Capture analysis and function lifting
   exist (`compiler/lower/lower.go:572-648`), but the closure is returned as
   an environment struct only, not paired with a function pointer. No program
   using closures has ever produced a working binary.

2. **Drop/destructors** — Types with `Drop` implementations should have
   `TypeName_drop(&_lN)` emitted in generated C (`compiler/codegen/emit.go:
   252-258`). The `InstrDrop` emission exists but no e2e test verifies that a
   destructor actually runs at the right time.

3. **Tuple construction and field access** — `let p = (1, 2); let x = p.0;`.
   Lowered to `InstrTuple`, emitted as anonymous struct. No e2e test.

4. **Struct initialization** — `Point { x: 1, y: 2 }`. Lowered to
   `InstrStructInit`, emitted as typed aggregate. No e2e test.

5. **Ownership: ref/mutref/owned/move** — Borrow lowering to `InstrBorrow`
   exists with `BorrowShared` and `BorrowMutable` kinds. No e2e test verifies
   that borrows produce correct C pointer semantics.

6. **Trait method dispatch** — `lookupMethod` in
   `compiler/check/methods.go:57-104` resolves methods through trait impls and
   supertraits. No e2e test compiles a program with a trait impl and calls a
   trait method on a concrete type.

7. **Implicit mutref on method receivers** — The language guide contract says
   `items.push(1)` should not require `mutref items` if `items` is a mutable
   binding. No e2e test verifies this.

8. **Loop with break value** — `let x = loop { break 42; };`. The lowerer
   captures break values via `BreakLocal`. No e2e test.

## Category 2: Partially implemented (code exists but incomplete)

9. **for..in loops** — Parsed and lowered, but the binding variable type is
   `Unknown` (`compiler/lower/lower.go:440`) and the iterator protocol is
   stubbed: the lowerer branches on the iterable directly instead of calling
   `next()` (`lower.go:451`). No program using `for x in collection` works.

10. **Optional chaining (?.)** — Parsed and type-checked, but returns
    `Unknown` type (`compiler/check/expr.go:312-316`). Lowering not
    implemented. No e2e test.

11. **Spawn/threads** — `spawn expr` lowered to `fuse_rt_thread_spawn` call
    but with `Unknown` typed locals (`compiler/lower/lower.go:133-143`).
    Runtime C function exists (`runtime/src/thread.c`). No e2e test.

12. **Channels** — `Chan[T]` struct declared in `stdlib/full/chan.fuse` with
    stub method bodies (`send` returns `Ok(())`, `recv` returns
    `Err("channel empty")`). No runtime integration. No e2e test.

13. **Pattern matching: struct and tuple patterns** — `StructPat` and
    `TuplePat` are parsed but have no lowering code. Only `WildcardPattern`,
    `BindPattern`, `LiteralPattern`, and `ConstructorPattern` are lowered.

14. **Unsafe blocks** — Parsed as regular blocks with no semantic enforcement.
    No compile error on unsafe FFI calls outside `unsafe { }`. No e2e test.

15. **Drop on nested/compound types** — `InstrDrop` only calls destructors on
    top-level locals. Struct fields with Drop implementations are not
    recursively destroyed. Moved-from fields are not skipped.

16. **Generic type inference from context** — `let r = Ok(42)` infers
    `Result[I32, Unknown]` instead of `Result[I32, Bool]` when later used as
    `try_get[I32, Bool](r)`. Only explicit type args on helper functions
    work around this.

## Category 3: Specified in language guide but not implemented

17. **Const declarations** — `const N: I32 = 42;` is parsed
    (`compiler/parse/item.go`) but no const evaluation or const-to-literal
    propagation exists. The checker visits `ConstDecl.Value` but does not
    intern the result.

18. **Type aliases** — `type Id = U64;` is parsed but never resolved by the
    checker. Using a type alias would produce a struct type with the alias
    name instead of the underlying type.

19. **Where clauses** — `fn foo[T]() where T: Display` is parsed but the
    constraint is never checked. Generic type params can be used without
    satisfying any declared bounds.

20. **Trait bounds on generic params** — `fn foo[T: Display](x: T)` is parsed
    but the `Display` bound is not enforced during type checking. Any type
    can be passed regardless of bounds.

21. **@value struct decorator** — The language guide describes `@value struct`
    as a struct that auto-derives core traits. The decorator is parsed but
    no auto-derivation occurs.

22. **@rank(N) lock ordering** — The language guide describes static lock
    ordering annotations. Parsed but not enforced.

23. **Module visibility (pub)** — `pub` is parsed on items but no
    public/private enforcement occurs across module boundaries. All symbols
    are accessible if resolved.

24. **Associated types in traits** — The language guide mentions associated
    items in trait declarations. Not implemented.

25. **Trait default method implementations** — The `Default` trait exists in
    `stdlib/core/traits.fuse` but there is no mechanism for a trait to
    provide a default method body.

26. **Generic impl blocks** — `impl[T] Trait for Container[T]` is not
    handled by the specializer or checker. Only concrete impls work.

27. **Recursive type detection** — The checker does not detect infinite-size
    types like `struct Node { next: Node }`.

28. **String escape sequences** — The language guide specifies `\n`, `\r`,
    `\t`, `\\`, `\"`, and Unicode escapes. The lexer may not implement the
    full escape suite. No e2e test verifies escape handling in emitted C.

## Category 4: Standard library gaps

The stdlib consists of two tiers: `stdlib/core/` (OS-free) and `stdlib/full/`
(hosted). Every method body below would fail if compiled through the current
pipeline because they depend on features from Categories 1–3.

29. **stdlib/core/option.fuse** — `impl Option` declares `is_some()`,
    `is_none()`, `unwrap()`, `unwrap_or()`, `map()`. The `map()` method
    takes `f: Fn` (unparameterized function type). The impl block is not
    generic (`impl Option` not `impl[T] Option[T]`). These methods have
    never been compiled.

30. **stdlib/core/result.fuse** — Same pattern as Option: `impl Result`
    declares methods that use pattern matching on `self`, but the impl is
    not generic and `map()` takes an unparameterized `Fn`.

31. **stdlib/core/string.fuse** — `String` struct with `toUpper()`,
    `toLower()`, `len()`, `is_empty()`, `as_bytes()`, `contains()`, etc.
    Method bodies return `self.clone()` (stub). `clone()` is not
    implemented. `as_bytes()` returns `[U8]` slice but slice infrastructure
    is incomplete.

32. **stdlib/core/collections.fuse** — `List[T]`, `Map[K, V]`, `Set[T]`
    declared with method stubs. `List.push()` body is empty.
    `Map.insert()` returns `Option` without a type parameter. None of these
    can compile.

33. **stdlib/core/traits.fuse** — Core traits (`Equatable`, `Comparable`,
    `Hashable`, `Display`, `Debug`, `Default`, `Clone`, `Drop`, `Iterator`,
    `IntoIterator`) are declared. `Iterator.next()` returns `Option` without
    a type parameter. `IntoIterator.into_iter()` returns `Self`. None have
    implementations on any concrete type.

34. **stdlib/core/hash.fuse** — Hash module for the Hashable trait surface.
    Not compiled or tested.

35. **stdlib/core/primitives.fuse** — Declares type aliases (`type Int =
    ISize`, `type Float = F64`) and primitive method stubs. Type aliases are
    not implemented (Category 3, item 18).

36. **stdlib/full/io.fuse** — `print()`, `println()`, `eprint()`,
    `eprintln()` call `fuse_rt_io_write_stdout/stderr` via extern FFI inside
    `unsafe` blocks. `File` struct with `open()`, `read()`, `write()`,
    `close()`. These depend on: String.as_bytes() (broken), unsafe blocks
    (unenforced), extern FFI (works), Result[T, E] (works after Wave 17).

37. **stdlib/full/chan.fuse** — `Chan[T]` with stub method bodies. `send()`
    returns `Ok(())`, `recv()` returns `Err("channel empty")`. No runtime
    queue integration.

38. **stdlib/full/thread.fuse** — `thread_spawn()` calls
    `fuse_rt_thread_spawn` via FFI. Depends on closure-to-function-pointer
    conversion which is incomplete.

39. **stdlib/full/sync.fuse** — Mutex, Cond with extern declarations for
    runtime sync primitives. Not compiled or tested.

40. **stdlib/full/os.fuse** — `argc()`, `argv()`, `env_get()`, `exit()` via
    runtime FFI. Not compiled or tested.

**Cascading effects**:
Wave 18 (Retirement of Go and C) cannot proceed until every feature specified
in the language guide has a proof program. The standard library must compile
through the pipeline and its methods must produce correct behavior. A new wave
or set of phases must be inserted between Wave 17 and Wave 18 to close these
gaps systematically.

The following dependency chains determine the implementation order:
- I/O (print/println) depends on: String, slices, unsafe blocks, extern FFI
- Iteration (for..in) depends on: Iterator trait, generic impls, closures
- Collections (List, Map, Set) depend on: generics, iteration, traits, Drop
- Channels depend on: generics, threads, runtime integration
- Self-hosting (Stage 2) depends on: all of the above

**Architectural lesson**:
A language is not complete when its grammar is parsed. A language is not
complete when its type checker runs. A language is complete when its standard
library compiles, its proof programs run, and a user can write a non-trivial
program using documented features and get correct behavior. Every feature in
the language guide that lacks a proof program is an untested claim.

**Verification**:
This entry is verified when every item above (1–40) either has an e2e proof
program that passes, or has been explicitly descoped from the language guide
with a rationale. The minimum bar for Wave 18 readiness is: closures, I/O
(print/println), iteration (for..in), trait dispatch, and Drop must all work
end-to-end.