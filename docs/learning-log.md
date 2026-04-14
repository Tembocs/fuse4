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