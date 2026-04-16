# Fuse: Accumulated Lessons

> Bridge document. Distills everything the project has learned across
> four production attempts, twenty-three learning-log entries, and the
> Wave 18 Phase 11 stdlib-integration spiral into one readable place.
> The philosophy and pillars are unchanged. The doctrine for how work
> is scheduled and how contracts are written has evolved substantially.
>
> Anyone picking up Fuse should read this document first, then the
> foundational docs under `docs/`.

---

## 1. Philosophy (unchanged)

Fuse is a compiled systems language designed around four commitments:

1. **Memory safety without a garbage collector.** Ownership, deterministic
   destruction, explicit borrowing. No tracing collector, no hidden
   allocator runtime model for ordinary code.
2. **Concurrency safety without a borrow checker.** Channels as the
   primary message-passing primitive, ranked synchronization for shared
   state, `spawn` as the explicit concurrency construct. No `async`/
   `await`.
3. **Effects visible at the call site.** Mutation, fallibility, and
   unsafe behavior do not hide behind ordinary-looking calls. Cost and
   consequence are legible without knowing hidden context.
4. **Developer experience as a first-class constraint.** Diagnostics,
   bootstrap simplicity, reproducibility, and the rule that reading a
   function should tell you what it does — these are architectural
   concerns, not polish.

These four commitments constrain every other decision in the language.
When an implementation choice violates one of them, the implementation
is wrong, not the commitment.

---

## 2. Pillars (unchanged)

The commitments above are realized through five concrete language
mechanisms:

1. **Ownership and deterministic destruction.** Every value has exactly
   one owner. Destruction runs at the end of the owning scope. `Drop`
   is a trait, not a runtime hook.
2. **Explicit borrowing — `ref` and `mutref`.** Shared and mutable
   borrows are surface-visible at the use site. A borrow never outlives
   its owner. A borrow is never silent.
3. **Explicit error propagation — `Result`, `Option`, `?`.** Absence
   and failure are types in the type system, not control-flow
   primitives. `?` is a lowering contract, not a macro. There is no
   nullable reference and no `null` pointer.
4. **Channels and ranked synchronization.** `Chan[T]` is the primary
   cross-thread communication channel. `Shared[T]` protects shared
   mutable state. `@rank(N)` prevents lock-ordering mistakes at
   compile time. `spawn` is the only way to create a thread.
5. **Explicit unsafe boundary.** `unsafe { }` blocks are visible at
   the call site. Raw pointer operations, extern FFI, and runtime
   intrinsics live behind this boundary and nowhere else. Safe APIs
   do not smuggle unsafe behavior.

These five pillars are load-bearing. Any proposed feature — type
inference, generic specialization, stdlib convenience, whatever — that
requires relaxing a pillar is rejected until the guide is explicitly
amended. The pillars do not bend to make implementation easier.

---

## 3. What went wrong across four attempts

Fuse is the fourth production attempt. The first three accumulated
architectural debt and were restarted. The fourth (`fuse4`) ran for
seventeen waves before hitting a cluster of integration failures that
made the root causes legible. Twenty-three learning-log entries record
the local lessons. This section collects them into themes.

### 3.1 Self-verifying plans are not verification (L013, L014)

**The pattern.** Proof criteria were written alongside the
implementation they were meant to prove. The fix landed, the proof was
added to justify it, and CI went green — regardless of whether the
feature actually worked. Features were described as "complete" when
only the structure was in place and the behavior was stubbed.

**The rule that emerged.**
- Every feature needs an end-to-end proof program written **before**
  the fix, captured in a failing (red) state, and committed with the
  failure output pasted into the commit message.
- After the fix lands, the proof must move from red to green. If its
  failure signature merely shifts without disappearing, the fix is
  incomplete.
- Stubs must emit diagnostics, not silent defaults. A silent stub is
  indistinguishable from a working feature to both the test suite and
  the user.

**Why this keeps coming up.** The pressure to close a wave incentivizes
self-verifying plans. The defense is procedural: proofs land red and
reviewed before the implementation does.

### 3.2 Permissiveness compounds (L021, Wave 18 Phase 11)

**The pattern.** A checker that silently accepts unresolved types
enables downstream code to be written against informal assumptions.
Stdlib bodies use `Ptr.null()` that the checker never blesses because
the checker returns `Unknown` and accepts everything that's assigned to
it. User programs reference stdlib types without imports because the
checker synthesizes phantom struct entries under the current module.
Eighteen months later, the permissive defaults are load-bearing.

**The concrete cost.** The stdlib-integration crisis (L021) took
eighteen commits of escalating band-aids before the three actual root
causes were identified: no stdlib auto-load, generic templates leaking
into C output, module identity mismatch for generic instantiations.
Each band-aid fixed one symptom and exposed the next.

**The rule that emerged.**
- Silent fallbacks to phantom types are forbidden. Unresolved names
  are diagnostics, not synthesized struct entries.
- Unknown types do not cross into codegen. If a type is unresolved
  after checking and lowering, the compiler stops.
- Workarounds in one layer to compensate for a gap in another layer
  are forbidden. Fix the gap.

**The deeper lesson.** Rule 3.9 and Rule 6.9 exist because permissive
acceptance, once relied on, becomes informally specified behavior.
Being strict from day one is cheaper than being strict after years of
code has been written against permissive defaults.

### 3.3 Expedient workarounds accumulate at every layer

A follow-up to 3.2, observed during Section 5 of Phase 11:

**The pattern (at the type system).** To let `List.new()` assign to
`List[Entry]` without real generic inference, `isAssignableTo` was
relaxed to accept any same-(Kind, Name, Module) match regardless of
TypeArgs. Variant lookup was widened to a graph-wide search so
specializations placed in stdlib modules could resolve user types.
Struct-lit typing consumed a `currentExpected` channel to work around
missing bidirectional inference.

**Each one was shippable. Together they form the next L021.** Each
expedient workaround is a deferred language-design decision. The real
features — generic inference, pattern-based type inference,
bidirectional checking, imported-variant visibility — remain
unspecified. When the next integration wave happens, the workarounds
will be load-bearing and the real features will have to be designed
around them.

**The rule that emerged.**
- When an implementation needs a feature the specification does not
  have, stop. Either update the specification or change the
  implementation. Do not invent informal language behavior under the
  radar of the guide.
- A "temporary" permissive check in the type system is a permanent
  permissive check until the guide says otherwise.

### 3.4 Contracts before code (L002, L021, L022)

**The pattern.** Contracts — how two components interact, what
invariants hold at a boundary — were often written after the
components were built. The stdlib was written before the auto-load
contract existed (L021). The codegen was written before the Ptr null
contract existed (L022). The type checker was written before the
variant-visibility contract existed (Section 5).

**Why "after" doesn't work.** When a contract is written after the
components, it has to be backward-compatible with what each component
already does. The contract becomes a description of current behavior
rather than a decision about correct behavior. Mistakes in current
behavior become codified.

**The rule that emerged.** A wave cannot close until the contracts
downstream waves will depend on are **specified and property-tested**,
not just implemented. The contracts belong in the language guide's
"Implementation contracts" section with named invariants and
reproducible test corpora, before any dependent code is written.

### 3.5 Stdlib is the stress test (L002, L018, L021)

**The pattern.** Stdlib was treated as downstream code that exercises
the compiler. In practice, stdlib is the first large corpus where the
compiler's contracts meet real usage shapes: generic specialization at
multiple depths, tagged enums with heterogeneous payloads, heap
allocation patterns, trait method dispatch across modules,
borrow-of-borrow patterns, match bodies with branching locals.

**The correct reading.** Stdlib is the compiler's first serious test.
If a stdlib module fails to integrate, the compiler is wrong — not the
stdlib — until proven otherwise.

**The rule that emerged.**
- Stdlib bodies are checked in the same pass as user code. No
  skipping.
- Stdlib integration — compiling a user program that uses stdlib
  types — is proven end-to-end as early as possible, not deferred to
  a late phase.
- A stdlib-shaped property-test corpus exists for every backend
  representation contract (pointer handling, enum layout, liveness
  across joins, runtime ABI shape) and runs before the stdlib bodies
  that depend on it are written.

### 3.6 Backend representation is architecture, not cleanup (L004, L005, L008, L022)

**The pattern.** Backend representation decisions — how `Ptr[T]` is
laid out in C, how tagged enums are represented, when unit types are
erased, how monomorphization is sequenced relative to ownership — were
treated as codegen-local concerns. In practice they are language
semantics: they decide what valid Fuse programs mean at the machine
level.

**The concrete examples.**
- Pointer categories (raw Ptr vs. ref/mutref borrow) must be distinct
  at the MIR level, not just at the surface (L004).
- Unit erasure must be total and global — a half-erased Unit corrupts
  call signatures (L005).
- Monomorphization must complete before codegen — a generic-parameter
  TypeId reaching codegen produces `Fuse_*__T` symbols that do not
  exist (L008).
- Tagged enum layout for heterogeneous variant payloads needs a C
  union keyed by `_tag`, not sequential slots — the sequential-slot
  layout works for Option-shaped enums and breaks the moment Result
  with T ≠ E appears (L022).
- Pattern matching dispatches on discriminants, not by falling through
  test chains — the fall-through shape produces wrong code for
  exhaustive matches (L007).

**The rule that emerged.** Rule 3.10: backend contracts are
architecture, not cleanup. Every backend representation decision is
specified in the language guide, property-tested with real-usage
shapes, and frozen before stdlib depends on it.

### 3.7 Language design must not serve implementation expedience

**The pattern.** When a stdlib author needed a way to express "List
hasn't allocated yet," they reached for `Ptr.null()`. The compiler
accepted it because the checker was permissive. It was never in the
language guide. It never had a contract. It contradicts Pillar 3
(no nullable reference, no null pointer). It is now entrenched across
multiple stdlib modules.

**Why it's corrosive.** An unsanctioned language feature introduced
for implementation convenience:
- violates the language philosophy without a design conversation,
- creates downstream code that depends on the unsanctioned feature,
- becomes difficult to remove because removing it breaks working code,
- trains contributors to reach for similar unsanctioned features.

**The rule that emerged.** When the stdlib (or any dependent layer)
needs a feature the language does not have, the options are:
1. Change the design: add the feature to the guide, with full
   specification and contract.
2. Change the implementation: find a way that uses only sanctioned
   features.

There is no third option. "We'll make it work and document it later"
is the failure mode.

**Concrete correction.** `Ptr.null()` is not in the guide and
violates Pillar 3. It is removed in the next rewrite. Heap-allocated
collections express "not yet allocated" through one of:
- a dangling-but-non-null pointer (typed-aligned sentinel, never
  dereferenced; guarded by `len == 0`),
- `Option[Ptr[T]]` at the data-field level (absence is explicit in
  the type system),
- eager allocation of a zero-length block at `new()` time.

The decision between these three is a language-guide decision, made
once, before the collection stdlib is written.

### 3.8 Liveness and control-flow invariants need adversarial tests

**The pattern (L022 subset).** The liveness pass was implemented
against the control-flow shapes the then-available tests exercised —
mostly linear flow with shallow `if`/`match`. Stdlib bodies with
branching locals used after match joins hit a gap: locals born in one
arm and used after the join were not declared, producing undefined-C
references.

**The rule that emerged.** Each IR pass has a named invariant (what
property it preserves) and a property-test corpus built from the
shapes real usage will exercise — not toy programs. A pass cannot be
considered complete until the corpus exhaustively covers the
invariant.

### 3.9 Wave structure reflects dependencies, not convenience

**The pattern.** Waves were often ordered by implementation ease
rather than by the dependency graph. Concurrency was deferred past
type checking even though concurrency rules (Send/mutref/spawn) are
type-system constraints. Monomorphization was placed after ownership
analysis even though ownership-on-generic-params is intrinsically
ambiguous. Stdlib body implementation began before backend
representation contracts were frozen.

**The rule that emerged.**
- Concurrency semantics are part of the type system and are specified
  before the checker runs.
- Monomorphization runs before ownership analysis so ownership only
  sees concrete types.
- Runtime ABI is specified before codegen so codegen becomes a
  translation, not a set of creative choices.
- Stdlib body implementation begins only after every contract it
  depends on is pinned.
- The wave order is dictated by dependency direction. If a proposed
  order requires deferring a contract that a later wave will need,
  the order is wrong.

### 3.10 Exit criteria are behavioral, not structural (L014, L016, L017, L020)

**The pattern.** Wave exit criteria often read "HIR nodes carry
metadata fields," "MIR terminates correctly," "all type expressions
resolve." Each is necessary. None proves the feature works for a
user. The L016 audit found 40 features with passing structural tests
and no behavioral proof.

**The rule that emerged.** Every wave's exit criteria include at
least one behavioral requirement: a program that uses the feature
compiles, links, runs, and returns an expected exit code (or produces
expected stdout). Structural criteria are necessary; behavioral
criteria are what closes the wave.

### 3.11 Determinism is architectural (Rule 7)

**The pattern.** Early code relied on Go's map iteration order for
"consistency" across runs. It's not consistent. Golden tests that
encoded iteration orders became flaky. Bootstrap tests that hashed
generated C output failed intermittently.

**The rule that emerged.**
- Same input, same compiler version, same target → byte-equivalent
  output.
- Symbol mangling depends only on semantic identity, never on
  iteration order or timestamps.
- No ambient randomness in output-affecting paths.
- No timestamps or absolute paths in goldens.
- If unique names are needed, use deterministic counters.

### 3.12 The seven features every Fuse-like language needs early (L019)

Distinct from representation contracts, these are surface-grammar
features whose absence blocks everything downstream:

1. **Associated types in traits** — without them, no `Iterator`, no
   `for..in`, no trait method whose return type depends on Self.
2. **`for..in` iteration** — the primary loop construct; depends on
   the Iterator protocol.
3. **Optional chaining (`?.`)** — specified in the surface grammar;
   silent no-op if deferred.
4. **Where-clause enforcement** — without it, trait bounds are
   syntactic noise and generic specialization cannot be validated.
5. **Trait default methods** — without them, trait usability collapses
   and stdlib traits (Equatable, Comparable, Hashable) are noisy.
6. **Module visibility (`pub`)** — without enforcement, encapsulation
   is a convention, not a contract.
7. **Array literals and array types** — foundational data shapes;
   their absence drives ad-hoc workarounds in every user program.

**The rule that emerged.** These seven are scheduled as explicit,
early waves or phases. None is allowed to be "absorbed into other
work." Each gets its own contract, proof program, and exit criterion.

---

## 4. The doctrine that emerges

The lessons above converge on a small set of doctrines that govern
everything else.

### Doctrine 1 — Pillars are inviolable

The four commitments and five pillars are load-bearing. Any design
decision, stdlib implementation, or backend representation that
requires relaxing a pillar is rejected. The pillars do not bend.
Specifically: **there is no null pointer. Absence goes through
`Option[T]`. A stdlib or compiler-internal use of `Ptr.null()` is a
bug, not a feature.**

### Doctrine 2 — Contracts are deliverables

Every wave produces contracts, not just implementations. A contract
is:
- a named invariant in the language guide,
- a property-test corpus shaped like real usage,
- an exit criterion that cites both.

A wave cannot close until every contract that downstream waves will
depend on is specified, tested, and frozen. "Implemented" is not
"specified."

### Doctrine 3 — Strict by default, diagnostic on miss

The compiler is strict from the first wave. Unresolved types produce
diagnostics. Unknown struct literals produce diagnostics. Unimplemented
features produce diagnostics. Silent acceptance is forbidden at every
layer — checker, monomorph, lowerer, codegen. If a feature is not yet
implemented, it emits a diagnostic that says so.

### Doctrine 4 — Prove behavior, not structure

Every feature has an end-to-end proof program. Proof programs are
committed red first, move to green through root-cause fixes, and
never describe the implementation — they describe the behavior a user
will observe. Structural tests are necessary but never sufficient.

### Doctrine 5 — Workarounds compound

A workaround in one layer for a gap in another layer is rejected. Fix
the gap. If the gap cannot be fixed in this wave, stop and escalate
the wave structure. "We'll clean it up later" is the failure mode
that produced L021 and L022.

### Doctrine 6 — Stdlib proves the compiler

Compiling a user program that uses stdlib types end-to-end is proven
as early as possible. Stdlib body implementation begins only after
every contract it depends on is pinned. Stdlib failures indict the
compiler until proven otherwise.

### Doctrine 7 — Wave order follows dependency, not convenience

The wave order is dictated by the dependency graph. If a proposed
order requires deferring a contract that a later wave needs, the
order is wrong. Concurrency before type checking. Monomorphization
before ownership. Runtime ABI before codegen. Stdlib after every
contract it depends on.

### Doctrine 8 — Determinism is built in, not added

Reproducibility is an architectural constraint from Wave 00. Every
output-affecting path is deterministic. Every golden is byte-stable
across machines and days. Symbol mangling depends only on semantic
identity.

---

## 5. Forbidden patterns

Concrete shapes that must never appear. Each has a learning-log entry
explaining the specific bug that produced it.

- **Silent stubs.** A feature that parses and type-checks but lowers
  to no-op is forbidden. Stubs emit diagnostics. (L013, Rule 6.9)
- **Phantom type synthesis.** Unresolved type names do not synthesize
  `Fuse_<currentModule>__<name>` struct entries. They emit
  diagnostics. (L021, Rule 3.9)
- **`Ptr.null()` or any null pointer value.** The language does not
  have a null pointer. Absence is `Option[T]`. (Pillar 3, corrected
  in L022 follow-up)
- **Workarounds for another layer's gap.** Double-pointer fixes in
  codegen that compensate for wrong borrow lowering; permissive
  `isAssignableTo` that compensates for missing generic inference;
  graph-wide fallback lookups that compensate for incomplete variant
  visibility. (Rule 4.2, Section 5 retrospective)
- **Backend representation chosen without a contract.** Tagged-enum
  C layout, pointer cast insertion, unit erasure, monomorphization
  ordering — each is specified in the guide before codegen is
  written. (L022, Rule 3.10)
- **`coreTypeLookup`-style hard-coded fallback tables.** If a name
  does not resolve, diagnose. Do not add a canonical-module table
  that silently resolves it. (L021)
- **Self-verifying proof programs.** Proofs committed green alongside
  the fix. Proofs are written red first, committed failing, and move
  to green through the fix. (L013, L014, Rule 6.8)
- **Hidden unsafe behavior in safe APIs.** A safe API that internally
  performs unsafe operations without the boundary being visible is
  forbidden. (Pillar 5, Rule 12.3)
- **Ambient randomness in output-affecting paths.** Iteration orders,
  timestamps, wall-clock values, random IDs that affect mangling or
  emission are forbidden. (Rule 7.3)
- **Generic templates reaching codegen.** A function or type with a
  `KindGenericParam` transitively is a template. Templates never emit
  C. Only monomorphized specializations do. (L008, Rule 3.9)

---

## 6. How this document is used

- **Before starting a wave.** Read the relevant lessons. Confirm the
  wave's contracts are named and property-tested before
  implementation begins.
- **During a wave.** When the temptation arises to add a permissive
  fallback, a silent stub, or a workaround, stop. The pattern is
  listed in §5 for a reason.
- **At the close of a wave.** Re-read §4 (doctrines). Every doctrine
  should be satisfiable with concrete evidence from the wave's
  deliverables.
- **When the project spirals.** The spiral is almost certainly one of
  the patterns in §5 or a violation of a doctrine in §4. Identify
  which. Revert. Rewrite with the correct contract.

The goal is not to prevent mistakes. The goal is to prevent the
**same** mistakes Fuse has already made four times.

---

## 7. What survives the rewrite

1. **The four commitments** (§1).
2. **The five pillars** (§2).
3. **The three IRs** (AST → HIR → MIR; no skipping; no collapsing).
4. **The bootstrap strategy** (Go → C11 → self-hosted Fuse), with the
   Go+C11 retirement scheduled once Stage 2 is stable.
5. **The five foundational documents** under `docs/` (language guide,
   implementation plan, repository layout, rules, learning log), with
   the plan rewritten to reflect the new wave order.
6. **The learning log** itself. L000 through L022 stay. New entries
   continue the sequence.
7. **This document.** Updated as new lessons land, superseding
   nothing but growing with the project.

What does not survive: the existing wave schedule, the existing
stdlib body implementations that depend on `Ptr.null()` and the
sequential-slot enum layout, the existing isAssignableTo permissive
check, the existing Section 5 expedient workarounds. Those are the
debt the rewrite clears.
