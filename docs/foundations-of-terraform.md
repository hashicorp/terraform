# The Foundations of Terraform

### A Layered Account of the System's Semantics, and the Premises That Bind Its Extensions

**Status:** Draft
**Audience:** Terraform Core engineers, architects, and designers of Terraform-adjacent languages
**Classification:** Internal

---

## Abstract

Terraform is usually described feature by feature: resources, modules, state,
providers, the plan. This document describes it instead as a layered semantic
system, and argues that most of what Terraform does — and most of what it
refuses to do — follows from a small number of premises established at its
lowest layers.

Part I characterizes Terraform as a data-flow rather than control-flow language,
in which the author expresses relationships between values and the engine
derives execution order from references alone; and characterizes a Terraform run
as the construction of a plan, which is a binding proposal rather than a dry
run. Part II describes the three representational systems that make those
characterizations expressible: the schema-directed configuration language, the
value model — including unknown values, which are the mechanism that makes
planning possible and are deliberately unobservable to the program — and the
address algebra that distinguishes configuration objects from instances. Part
III describes the machinery: graph construction by transformation, instance
expansion, the resource lifecycle, the schema-mediated Core/provider contract
and the assertions that enforce it, state as a binding ledger rather than a
cache, and destroy as reversal. Part IV describes composition and extension:
module encapsulation, the inert refactoring constructs that rebind addresses
without touching remote objects, the family of derived HCL languages, and
compatibility as a semantic property of the system rather than a release
policy.

Twenty-four load-bearing claims are stated as numbered, citable items and
indexed in Appendix A, each tagged as an axiom, a derived guarantee, or a design
policy, with a support relation showing which guarantees fail if each is
relaxed. Appendix C registers the places where Terraform does not satisfy its
own premises, and what each exception costs.

The intended use is to make system-level objections in design review
*nameable*. A reviewer who observes that a proposal disturbs a lower-layer
premise has raised a sufficient objection by naming it and the guarantee at
risk; the burden then shifts to the proposer, who must show preservation,
coordinated replacement of the premise and all its dependants, or explicit
quarantine of the weakened guarantee behind a boundary its consumers can see.
What is not available is the option design discussions default to: weakening a
guarantee silently.

---

## 0. How to Read This Document

### 0.1 Thesis

Terraform's foundations are **layered**. Each layer supplies guarantees to the
layer above it, and each layer's guarantees are *purchased* by the invariants of
the layer beneath it. The address algebra is meaningful only because evaluation
is pure; the dependency graph is correct only because references are the sole
source of ordering; the plan is trustworthy only because the value model can
represent the absence of knowledge honestly.

From this follows the document's central claim, which governs how it should be
used in design review:

> **Identifying that a proposal disturbs a lower-layer premise is, by itself, a
> sufficient objection to require an answer.** The objector's obligation is to
> name the premise and the dependent guarantee at risk. The burden then shifts to
> the proposer.

The failure is a *consequence* of the violation, not a hypothesis about it, and
the proposer — not the reviewer — must discharge it. There are exactly three
ways to do so:

1. **Preservation.** Show that the premise in fact still holds, and that the
   apparent disturbance is not one.
2. **Coordinated replacement.** Show that the premise can be replaced by a
   different one, *and* enumerate every dependent guarantee, showing what
   becomes of each. This is a much larger claim than it appears, because the
   dependants are rarely local to the feature under discussion.
3. **Explicit quarantine.** Confine the weakened guarantee to a separately
   identifiable runtime, artifact, edition, or opt-in mode, such that consumers
   who relied on the stronger guarantee are not silently served the weaker one.

The third route is real and has been used deliberately — deferred changes
(§8.3), Stacks (§15.2), and the language editions mechanism (§16.4) are all
instances of it. What is *not* available is the fourth option that design
discussions gravitate toward by default: weakening a guarantee silently, so that
downstream consumers continue to assume a promise that no longer holds.

This inversion of burden matters because the alternative does not scale. No
individual can hold the entire interaction surface of Terraform in working memory
well enough to derive, on demand, the specific future contradiction that a
premise violation will produce. Requiring that derivation before an objection can
be sustained systematically biases decisions toward violation, because the cost
of proving harm is borne by the objector while the cost of the harm is deferred
onto everyone. Naming premises explicitly, and treating them as citable, is what
makes system-level reasoning tractable for a group rather than a heroic act by an
individual.

### 0.2 Three Kinds of Claim

The numbered items in this document are not all the same kind of thing, and
treating them as uniformly absolute is the most likely way to misuse it. Each is
tagged in Appendix A as one of:

- **Axiom (A).** An assumption the rest of the system is built on. Disturbing one
  requires the full discharge above. P1, P2, P3, P7, P9, P10, P11, P13, P15, P16,
  P17, P23 are axioms.
- **Derived guarantee (D).** A property that *follows* from axioms, and which
  therefore fails automatically if its premises fail. These are cited to identify
  *what breaks*, not as independent constraints. P5, P6, P12, P18, P24 are
  derived.
- **Design policy (P).** A strongly-held commitment that is nonetheless a choice,
  revisable with justification rather than requiring the discharge above. P4, P8,
  P14, P19, P20, P21, P22 are policies.

An objection citing an axiom is much stronger than one citing a policy, and a
reviewer should say which they mean.

### 0.3 Structure

The document proceeds in four layers, from the most abstract to the most
composed. **Part I (Paradigm)** establishes what kind of language Terraform is
and what a Terraform run fundamentally *is*. **Part II (The Substrate)**
describes the three representational systems on which everything else is
built: the configuration language, the value model, and the address algebra.
**Part III (The Machinery)** describes how those representations are turned
into action: the dependency graph, instance expansion, the resource lifecycle,
the provider contract, and state. **Part IV (Composition and Extension)**
describes how Terraform is scaled out — modules, refactoring, the derived
language family — and why compatibility is a semantic property of the system
rather than a release-management policy.

The ordering is not merely pedagogical. It is a dependency order. A claim made
in Part III is licensed by premises established in Parts I and II, and a
feature that disturbs those premises invalidates the Part III claim regardless
of whether the feature's author intended to touch it.

### 0.4 Notation and Conventions

Load-bearing claims are numbered **P1**, **P2**, … and given short names, so
that they can be cited by name in design review ("this disturbs P2, Reference
Order"). Each carries a tag — **(A)** axiom, **(D)** derived guarantee, **(P)**
design policy — per §0.2. A consolidated index appears in Appendix A. Claims are
stated about the *system*, not about any particular implementation of it; where
the current implementation deviates, that deviation is noted explicitly as a
**wart** rather than being silently normalized into the claim.

**Scope.** P1–P24 govern the Terraform configuration language and the Terraform
Core runtime that evaluates it. They do *not* automatically govern the derived
languages of §15: each of those either inherits, revises, or declares a given
premise inapplicable, and §15 says which where it is known. A design argument
about Stacks or Test must therefore establish that the premise it cites applies
there, rather than assuming it.

Citations to the implementation take the form `internal/path/file.go:line` and
refer to the `hashicorp/terraform` repository at the revision current with
Terraform v1.16. Citations to documentation refer to the v1.16 documentation
set. Because implementation details drift, code citations are evidence for
claims, not the claims themselves; where the code and the stated premise
disagree, the premise is the thing under discussion.

Throughout, **Core** means Terraform Core — the graph and evaluation engine in
`internal/terraform` and its supporting packages — as distinct from providers,
from Terraform CLI's command layer, and from any wrapping automation.

---

# Part I — Paradigm

Part I answers three questions: what kind of language Terraform is, what a run
of Terraform is, and what it means for a run to succeed. The premises
established here are the deepest in the system. They are also the least
frequently written down, which is precisely why they are the ones most often
disturbed by accident.

## 1. Terraform Is a Data-Flow Language

### 1.1 "Declarative" Is Not the Useful Distinction

The word most consistently applied to the Terraform language is "declarative,"
and it is the word that has produced the most unproductive argument. Declarative
programming describes a desired outcome rather than a process for achieving it,
and it is properly contrasted with *imperative* programming. It is not, as is
frequently assumed in design discussions, contrasted with *programming*:

> "Declarative language" is not the opposite of "programming language". In this
> context, "declarative" is better contrasted with "imperative", which is a more
> common programming paradigm where the programmer describes a sequence of steps
> that *imply* a particular result.
>
> — Martin Atkins, *Evolving the Terraform Language* (2019)

This matters because "is this declarative enough?" is not a question that can be
adjudicated. Appeals to the term have been used both to reject perfectly
well-founded features and to wave through ill-founded ones. A sharper and
falsifiable characterization is available, and it is the one this document
adopts.

### 1.2 Data Flow Rather Than Control Flow

General-purpose languages are predominantly **control-flow** languages: the
author writes a sequence of steps and inserts conditional jumps, producing a
control-flow graph that enumerates the possible execution paths through the
program. Terraform is a **data-flow** language: the author writes expressions
describing how data from one object is used to construct another, and the
result of evaluating the program is a *data-flow* graph.

> Data-flow programming […] is concerned with the movement of data rather than
> with an exact order of execution. In Terraform we write code that expresses
> the relationships between different objects — or, more specifically, how to
> use data from one object to construct another object. The result of evaluating
> a Terraform program is a data flow graph rather than a control flow graph.
>
> — Martin Atkins, *Terraform is a Data Flow Language* (2019)

This is the correct frame for evaluating a proposed language feature. The
question is not whether a construct "feels declarative" but whether it
introduces control flow — that is, whether it gives the author a mechanism to
determine *when* something happens other than by expressing what data it
depends on.

By this standard, the constructs added in Terraform v0.12 that were widely
alleged to be a retreat from declarativity are nothing of the sort. Conditional
expressions, `for` expressions, `dynamic` blocks, and `for_each` are all
data-flow constructs: each is a more general way of combining values to produce
values, or of projecting nested-block structure from values. None of them lets
the author sequence operations. The decisive property they share is that
although they *imply* an ordering, the ordering is chosen by the engine:

> What they all have in common is that none of them introduce control flow:
> while these constructs do *imply* a sequencing of operations, the evaluation
> order is ultimately decided by the Terraform language engine rather than the
> Terraform module author.

This yields the first and deepest premise of the system.

> **P1 (A) — Data Flow.** A Terraform configuration expresses relationships between
> values, not a sequence of operations. The author describes *what depends on
> what*; the engine decides *when*. No language construct may give the author
> direct control over sequencing.

And its operational corollary, which is the single most-violated premise in
practice because it is the one that feels like an implementation detail rather
than a semantic commitment:

> **P2 (A) — Reference Order.** Evaluation order is determined solely by
> references. If A must be evaluated before B, that fact must be recoverable from
> a reference appearing in B. Ordering established by any other means does not
> exist as far as the rest of the system is concerned.

A **reference**, in the sense P2 requires, is an occurrence in the configuration
of a `Referenceable` address (§6.4). It is *not* the same thing as a value
dependency, and conflating the two is the most common source of confusion about
this premise. A reference may be *traversed* to produce a value —
`aws_instance.example.id` refers to `aws_instance.example` and then reads an
attribute of it — but it need not be. `depends_on = [aws_instance.example]` is
equally a reference: it names a `Referenceable` address and produces an edge,
and it simply yields no value. What makes something a reference is that it
points at an address the language can resolve, not that data flows along it.

This distinction disposes of an objection that is otherwise natural. Terraform's
graph builders contain several transformers that add edges —
`AttachDependenciesTransformer`, `DestroyEdgeTransformer`, `CBDEdgeTransformer`
(§7.1) — and a reader may conclude that these are independent sources of
ordering that falsify P2. They are not. Each is a *derivation over* the
reference-derived structure: `CBDEdgeTransformer` inverts edges that references
already established, `DestroyEdgeTransformer` computes the destroy-side
consequences of the same dependency data, and recorded state dependencies
(`internal/states/instance_object_src.go:24-78`) preserve references for objects
whose configuration no longer exists so that the structure survives the
declaration. None of them lets an author order two objects that do not refer to
one another.

P2 is not a statement about the graph builder. It is a statement about what the
graph builder is *allowed to be asked to do*. The clearest articulation of why
comes from Terraform Core itself:

> the declarative nature of Terraform (essentially a dataflow language) requires
> that we be able to evaluate all references statically in order to build the
> graph before the full evaluation can begin. You can think of this as
> compilation in a static programming language, where all symbols need to be
> valid in order to compile.
>
> — James Bardin, `hashicorp/terraform#21953` (2023-02-08)

The graph is built *before* evaluation, and it is built from references. Every
subsystem downstream —
dependency resolution, destroy-order reversal, `create_before_destroy`
propagation, transitive reduction, targeted operations, refactoring, and the
detection of dependency cycles — is written against the assumption that the
reference set is the complete ordering specification. A feature that establishes
an ordering relationship by some other channel does not merely add an
unmodeled edge; it makes every one of those subsystems' guarantees unsound,
because each of them reasons about orderings by reasoning about references.
The unsoundness is general, and a design review need not exhibit the particular
future contradiction it will produce (§0.1).

The practical test P2 supplies is therefore sharp, and it is not "does data flow
between these objects?" It is: **can the ordering this feature requires be named
as a reference from one address to another?** An ordering expressed as a
position in time — *before* this, *after* that — rather than as a pointer to an
address is outside the mechanism, however reasonable it looks in isolation.

### 1.3 Terraform Is Not Alone in This

Data-flow programming is not a Terraform idiosyncrasy, and it is useful in
design discussions to have the other examples at hand, because they make the
constraints feel less arbitrary. Spreadsheets are data-flow: a cell's formula
consumes other cells, and the recalculation order is the spreadsheet's problem,
not the author's. Verilog and VHDL are data-flow, and arguably the purest form
of it — the computation is fixed in place as a circuit, and the data is what
moves. Futures and promises introduce data-flow into control-flow languages,
letting a runtime decide execution order based on when values become available.
Elm and functional reactive programming propagate change through a graph of
expressions. GStreamer models media processing as a pipeline graph whose data
propagation is the framework's responsibility.

In every case the same bargain is struck: the author gives up control over
sequencing, and receives in exchange the engine's ability to parallelize, to
reorder, to reverse, and to reason globally about the program. Terraform's
ability to destroy infrastructure in correct order, to apply unrelated changes
concurrently, and to tell you what it is going to do before it does it are all
purchased with exactly that currency.

### 1.4 The Absence of Conditional Existence Is Deliberate

A recurring request is for a configuration to branch on runtime-discovered
facts — most commonly, to check whether an object exists and create it only if
it does not. Terraform deliberately does not support this, and the reason is
instructive because it shows a premise being defended against a sympathetic
use case.

Such a thing *could* be expressed within data flow: the presence or absence of
the object would simply be another datum. The objection is not
paradigm-purity but consequence. `data` blocks are, by design, not merely a
retrieval mechanism but an *assertion*:

> Instead, `data` blocks in Terraform serve a dual purpose both as a way to
> refer to external objects from a Terraform configuration *and* as a
> declaration of the assumption that the external object exists […] If you make
> a mistake and apply the configurations in the wrong order, Terraform will
> report that the network doesn't exist yet, rather than having the network end
> up owned by some other module.

The failure mode being prevented is an *ownership* failure — two configurations
each concluding they should create the same object — and it is prevented by
making absence an error rather than a value. This is a preview of a theme that
recurs at every layer: Terraform repeatedly chooses to make an ambiguous
situation illegal rather than to assign it a default meaning, because a default
meaning would be a contract (§16).

The closely related request — a meta-argument that would disable a resource
outright, so that references to it need not change — has been declined for a
reason that follows directly from P2, and the argument is worth reproducing
because it is a rehearsal rather than a prediction. Because a disabled resource
"is just like commenting it out," one can simulate the feature today by actually
commenting the resource out; the result is that every reference to it becomes a
static error, and the only way to make the configuration valid again is to
delete those references — including in the case where the resource *is* enabled.
Conditional existence and dynamic referenceability are therefore mutually
incompatible requirements, not two features that happen not to have been built
yet.

This is also why `count = 0` and an empty `for_each` produce an *empty
collection* rather than nothing at all:

> `count` and `for_each` both handle the "nothing" case by becoming an empty
> collection […] This means that the resource value is still available to use, so
> you can ask questions like: what is the length of this collection? Does this
> map have a key called "foo"? I want to be clear that making e.g.
> `aws_instance.foo` be totally unavailable does not offer a comparable
> capability, because there would be no value there to evaluate conditions
> against.
>
> — Martin Atkins, `hashicorp/terraform#21953` (2022-10-28)

The much-maligned `count = 0` idiom is thus not a workaround around a missing
feature. It is the shape the feature has to take in a language where references
must resolve statically: absence is represented as an empty *value*, so that it
remains referenceable.

### 1.5 The v0.12 Design Principles

Three principles were adopted to arbitrate language design during the v0.12
redesign, and they remain the operative tie-breakers:

1. **Prioritize the reader over the writer.** Code is read far more often than
   written, and outlives its author.
2. **Explicit is better than implicit.** You should not need to be a Terraform
   expert to understand the intent of a configuration someone else wrote.
   Implicit behavior is obvious only to those who already know it is there.
3. **Simple things should be simple, and complex things should be possible.**

These are preferences, not premises: they resolve ties, and they are
occasionally overridden by pragmatism. The generalized splat operator is the
canonical acknowledged override — retained and regularized despite arguably
failing the explicitness test, because it was too well established to remove.
The distinction matters. Violating a preference requires a justification.
Violating a premise requires a redesign.

## 2. A Run Is a Proposal and Its Execution

### 2.1 The Plan/Apply Split

A Terraform operation has side effects — creating, updating, and destroying
remote objects — but those side effects do not originate in the program. They
originate in the *plan*:

> A Terraform operation *itself* has side-effects (creating, updating, or
> deleting remote objects) but they come not from the program itself but from
> the *plan*. Terraform's plan step evaluates the input program […] to obtain a
> description of the desired result, and compares that with the saved state to
> find any differences. Terraform Core then, with the help of providers,
> generates a set of create, update, and delete actions which can be performed
> to converge on the desired result.

Evaluation of the configuration is therefore *pure*. Its output is a
description. The apply phase is the only thing that mutates the world, and it
operates on the description rather than on the configuration. This separation is
what makes a plan reviewable, storable, transmissible, and approvable by a
person or a policy engine that never runs the configuration itself.

> **P3 (A) — Pure Evaluation.** Evaluating a configuration produces values and a
> description of intent. It does not produce side effects. Any operation with
> externally-visible effects belongs to apply, and is reachable only through a
> plan.

This premise is stated normatively in Core's own design documentation:

> A key design tenet for Terraform is that any actions with externally-visible
> side-effects should be carried out via the standard process of creating a plan
> and then applying it. Any new features should typically fit within this model.
> There are also some historical exceptions to this rule, which we hope to
> supplement with plan-and-apply-based equivalents over time.
>
> — `docs/planning-behaviors.md`

The acknowledgement of "historical exceptions" is important and should be read
precisely. It is not a license. It is a statement that the exceptions are debt.

### 2.2 The Plan Is a Proposal, Not a Script

The plan is better understood as a *proposal* than as a program to be executed
blindly. Core proposes an action for each resource instance; the provider is
consulted and may modify the proposal; the operator may reject it. The
vocabulary of proposed actions is small and closed — Create, Read, Update,
Delete, Replace, No-op — and `Replace` is a meta-action that Core lowers into an
ordered pair of Create and Delete (`docs/planning-behaviors.md`).

The smallness of that vocabulary is load-bearing. Terraform's ability to reason
about arbitrary infrastructure rests on there being very few kinds of thing that
can happen to any of it:

> Terraform has a much smaller space of possible operations (create, read,
> update, delete) and we work at a higher level of abstraction where many
> separate smaller operations are grouped together into a single "action", which
> makes data-flow programming a more practical proposition for many cases.

> **P4 (P) — Closed Action Vocabulary.** The set of actions Core can propose for a
> managed object is fixed and small. Providers influence *which* action is
> proposed and *how* it is parameterized; they do not extend the vocabulary.

Deviations from default action selection are not ad-hoc. They fall into exactly
three design patterns, distinguished by *who* activates them:
**configuration-driven** behaviors, specified by a module author and therefore
inherited by every caller (`ignore_changes`, `replace_triggered_by`,
`create_before_destroy`, `moved`); **provider-driven** behaviors, activated by
fields in a provider's response to a planning request and therefore appearing
automatic to the user (requiring replacement, or normalizing a value so an
apparent change becomes a no-op); and **single-run** behaviors, set as plan
options by the operator for an exceptional circumstance (`-replace`,
`-refresh-only`, `-target`).

This taxonomy is a genuine design tool and should be used explicitly when
proposing a behavior. Each pattern carries a distinct cost. Configuration-driven
behaviors become part of a module's published interface. Provider-driven
behaviors are the least burdensome on users and, per Core's own assessment, the
most underused. Single-run behaviors are the most expensive of all, because
every wrapping UI and automation must independently expose them or else silently
deny their users access to part of Terraform:

> However, this design pattern has the disadvantage that each new single-run
> behavior type requires custom work in every wrapping UI or automaton around
> Terraform Core, in order provide the user of that wrapper some way to directly
> activate the special option, or to offer an "escape hatch" to use Terraform
> CLI directly and bypass the wrapping automation for a particular change.
>
> — `docs/planning-behaviors.md`

### 2.3 The Plan Is a Contract With the Operator

Because a plan is reviewed and approved before it is applied, apply must not do
things the plan did not disclose. This is not a quality goal but a semantic
requirement, and Core enforces it mechanically by checking the applied result
against the planned value and rejecting violations
(`internal/plans/objchange/compatible.go:29-43`).

> **P5 (D) — Value Fidelity.** Anything the plan states as known must hold after
> apply. Anything the plan leaves unknown must be resolved within the constraints
> the plan published for it.

> **P24 (D) — Plan Authorization.** Every externally visible effect of an apply
> must have been represented in the reviewed plan — its subject, its action
> class, and its existence. Apply may resolve values the plan left open; it may
> not act on a subject the plan did not name, or take an action the plan did not
> propose.

These two are separated because they fail separately, and because features tend
to weaken one while leaving the other intact. P5 is about *values*: it is what
`AssertObjectCompatible` enforces (§10.4), and it is what the provider contract
is written against. P24 is about *scope*: it is the guarantee an operator relies
on when approving a plan, and the one that automation around Terraform assumes
when it treats a plan as a reviewable changeset. A plan that is honest about
every value it mentions but silently acts on an object it never mentioned has
satisfied P5 and violated P24.

Deferred changes (§8.3) are precisely a deliberate, quarantined weakening of P24
with P5 left intact — which is why they must be opt-in and separately
represented rather than folded into the ordinary change set.

P5 and P24 are the reason the value model must be able to represent *the absence
of knowledge* honestly rather than guessing (§5.3), and why the machinery for
doing so is foundational rather than incidental.

## 3. Convergence and Its Limits

### 3.1 The Fixed Point

The intended shape of a Terraform run is convergence on a fixed point: apply a
plan derived from configuration C and prior state S, and the resulting state S′
should be such that planning C against S′ yields no changes. An empty second
plan is the observable signature of a correct run, and it is the property most
Terraform testing — including `terraform test` and provider acceptance testing —
is ultimately checking.

> **P6 (D) — Convergence.** A successful apply reaches a fixed point with respect to
> its configuration. A configuration that plans a non-empty change immediately
> after a successful apply indicates a defect, not a workflow.

The diagnostic value of P6 is high because the defect it indicates is almost
always locatable: a provider that fails to record what it actually created, a
schema whose type does not match what the remote API returns, a normalization
the provider performs but does not report, or a configuration value that is
genuinely not stable.

### 3.2 Where Convergence Is Bounded

P6 is a claim about Terraform's relationship to its own state, not a claim of
idempotence against the world, and the boundaries are worth stating because
conflating them produces bad designs.

Terraform does not observe the world continuously. Between runs, the remote
system may change for reasons Terraform did not cause; this is *drift*, and
detecting it is the job of refresh (§11.3), not a violation of convergence.
Terraform does not own everything it can see: convergence is scoped to the
objects bound in state, which is why the binding is the load-bearing concept
rather than the query (§11.1). Some remote systems are genuinely
non-deterministic or eventually consistent, and a provider is responsible for
presenting a stable view across that; where it cannot, the instability surfaces
as a perpetual diff, which is properly a provider defect rather than a Core
concern. And convergence is a property of *complete* applies: a partial apply,
whether caused by an error or by `-target`, leaves a state that is by
construction not a fixed point of the full configuration, which is precisely why
targeted operations are documented as an exceptional recovery tool rather than a
workflow.

### 3.3 The Scope of Part I

Three things have now been fixed, and everything in the remainder of this
document depends on them. Order comes from references and nowhere else (P1, P2).
Evaluation is pure and the plan is the sole route to effect (P3). The plan binds
apply (P5). The next question is what representations make these claims
expressible at all — which is the subject of Part II.

---

# Part II — The Substrate

Part I established what a Terraform program *means*. Part II describes the three
representational systems that make those meanings expressible: the configuration
language, the value model, and the address algebra. These are the layer that
design work most often treats as settled background, and they are consequently
the layer whose premises are most often disturbed without anyone noticing.

## 4. The Configuration Language

### 4.1 Two Layers, One Syntax

The Terraform language is a domain-specific language built on HCL, and HCL is
best understood as two languages that share a syntax: a **structural** language
of blocks and arguments, and an **expression** language of values and operators.

The relationship between these two layers has changed once, decisively. Through
Terraform v0.11, the structural layer was HCL and the expression layer was HIL,
a separate string-interpolation language. The seam between them was visible and
painful: constructing a list looked different depending on which language you
were in, and an interpolation that returned a non-string was, in Atkins' words,
"highly counter-intuitive." Terraform v0.12 merged them, so that one expression
syntax is used everywhere and string interpolation is needed only for actually
building strings.

The merge had a consequence that is still being paid for, and it is worth
stating precisely because it recurs whenever a new block type is designed:

> because braces are now used both for nested blocks *and* for map expressions,
> the new language must now be more particular about the distinction between
> arguments and nested blocks, where before the language would usually figure
> out what the user meant.

Terraform can no longer infer from syntax alone whether `foo { ... }` is a nested
block or an argument assigned an object value. The distinction is resolved by
**schema**, not by syntax.

> **P7 (A) — Schema-Directed Interpretation.** The meaning of a configuration body
> is not determined by its syntax alone. A schema is required to decide which
> names are arguments and which are block types, what types values must be
> converted to, and which structures repeat. Configuration cannot be fully
> understood without the schema that governs it.

P7 has a practical consequence that constrains a great deal of tooling: because
provider schemas are obtained from provider plugins, *nothing can fully
understand a configuration until providers are installed*. This is why
`terraform init` is a prerequisite for `validate`, why language servers must
resolve providers to give good diagnostics, and why any proposal to analyze
configuration "statically" must be explicit about which of the two analyses it
means — the schema-free structural one, which can see very little, or the
schema-directed one, which is not free of installation.

The schema machinery itself lives in `internal/configs/configschema`. A `Block`
has `Attributes` and `BlockTypes` (`internal/configs/configschema/schema.go:16-47`);
an `Attribute` carries a type or a nested object type, plus the flags
`Required`, `Optional`, `Computed`, `Sensitive`, `WriteOnly`, and `Deprecated`
(`:54-106`). Block nesting modes are `Single`, `Group`, `List`, `Set`, and `Map`
(`:136-186`). `DecoderSpec()` derives an `hcldec` specification from the schema
(`internal/configs/configschema/decoder_spec.go:76-168`), `ImpliedType()` derives
the cty type (`implied_type.go:11-145`), and `CoerceValue()` forces a decoded
value into that type, filling absent optional values with null
(`coerce_value.go:13-260`).

### 4.2 JSON Is Not a Second Language

Terraform's JSON syntax is a second *surface* for the same language, not a
second language. Anything expressible in native syntax must be expressible in
JSON, because JSON exists precisely so that configuration can be generated by
programs. Any feature whose design is expressible only in native syntax has
created a machine-generation hole, and this is a routine oversight because
designers naturally prototype in native syntax.

### 4.3 Static Evaluation Is a Narrow, Deliberate Exception

Almost all expression evaluation happens during plan, after providers and
modules are installed. A small amount must happen earlier, because it determines
*what gets installed*. Module `source` and `version` arguments are resolved
during `terraform init`, before the graph exists.

The resolution is deliberately impoverished rather than general: `source` and
`version` may reference only constant input variables and local values, and any
input variable used there must be declared `const = true`
(`docs/language/block/module.mdx:92-100, 399-401`). This is a textbook instance
of the closed-design preference. A general expression here would have been more
expressive and would have created an evaluation context that runs before
providers exist, before state is read, and before the graph is built — a second,
weaker evaluation semantics that every future language feature would have had to
be defined against twice.

> **P8 (P) — One Evaluation Semantics.** There is one expression language with one
> meaning. Where evaluation must occur in a restricted context, the restriction
> is imposed on *what may be referenced*, not on what expressions mean. A
> construct must not evaluate to one thing in one phase and another thing in
> another.

## 5. The Value Model

The value model is where Terraform's most distinctive semantics live, and it is
the layer whose premises are least visible from outside Core. Terraform values
are `cty` values (`github.com/zclconf/go-cty`), and cty was built for this
purpose.

### 5.1 Types

cty provides primitive types (`string`, `number`, `bool`), collection types
(`list`, `set`, `map`) whose elements share a single type, and structural types
(`object`, `tuple`) whose members may differ
(`docs/language/expressions/type-constraints.mdx:38-153`). `any` is not a type
but a placeholder to be resolved by unification (`:217-275`). Conversion is
delegated to cty's `convert` package throughout evaluation
(`internal/lang/eval.go:216,257`; `internal/lang/funcs/conversion.go:49-76`), and
unification via `convert.UnifyUnsafe` is what gives `list(any)` and friends their
behavior (`internal/lang/funcs/collection.go:150,350,374`).

There is exactly one numeric type, and it is arbitrary-precision. The reasoning
is a good example of designing for the actual problem domain rather than by
analogy to general-purpose languages:

> Integers and floating point numbers are used in different situations as a
> performance tradeoff in traditional software, but performance at that level is
> irrelevant in Terraform since any minor difference is dwarfed by the time
> spent waiting for remote APIs to respond to requests.

Because remote APIs *do* specify machine types, the number domain is a superset
of 64-bit integers and 64-bit floats, and range errors are detected at the point
of conversion — just in time to be sent to an API — rather than silently during
arithmetic.

### 5.2 Null Means Absence, Not Emptiness

cty distinguishes a null value of a type from an empty value of that type, and
Terraform relies on the distinction. `cty.NullVal(T)` is a typed absence marker
used throughout (`internal/plans/dynamic_value.go:24-76`), and it is carefully
distinguished from Go-level `cty.NilVal`, which means "no value at all" at the
implementation layer rather than "null" at the language layer.

The distinction is load-bearing at the provider boundary. Because null is
distinguishable from the empty string, zero, and the empty collection, a
provider can tell "the practitioner did not set this" from "the practitioner set
this to nothing" — a distinction that maps directly onto most remote APIs'
distinction between omitting a field and clearing it. Planning logic relies on
it: a planned null becoming a non-null actual value is an error
(`internal/plans/objchange/compatible.go:35-43`), and `ProposedNew` special-cases
null prior and config values specifically to avoid fabricating blocks that were
not requested (`internal/plans/objchange/objchange.go:15-31,75-80`).

The `nullable = false` argument on input variables exists so that a module author
can refuse null at the interface boundary rather than defending against it
everywhere downstream (`docs/language/block/variable.mdx:215-241`).

### 5.3 Unknown Values

An unknown value is a value that is known to *exist* and known to have a *type*,
but whose content is not yet determined. Unknowns are what make the plan/apply
split possible at all: without them, planning a configuration in which one
resource's argument comes from another resource's not-yet-created attribute would
have no representation except failure.

The framing that makes their role clearest is that planning is *compilation*:

> Rather than behaving as if it's taking the actions but just stubbing out the
> side-effects, Terraform's plan phase is in a sense *writing a program* to
> achieve the desired state. When you apply the plan, Terraform then *runs* that
> program, ensuring along the way that it'll either do what the plan said it
> would do or generate an error explaining why it can't.
>
> — Martin Atkins, *Unknown Values: The Secret to Terraform Plan* (2021)

Unknown values are the placeholders in that program. In the UI they appear as
`(known after apply)`; the two ideas are identical.

### 5.3.1 Unknowns Are Not Promises

The comparison to futures and promises is natural and is the most instructive
wrong answer in the subject, because the difference is precisely the property
Terraform depends on.

> The key difference between unknown values and promises is that unknown values
> are not part of the explicit programming model *at all*. Instead, they are
> hidden inside the language runtime and handled automatically as part of
> expression evaluation.

A promise is a value the program can *observe*: it can test whether the promise
is resolved and behave differently depending on the answer. That observability
destroys the plan guarantee, because a program that can see whether a value is
known can produce one result during plan and a different one during apply —
deliberately, or, far more commonly, by accident, as when a value is derived
from an API whose response changes between the two phases.

Terraform therefore denies the program any way to ask:

> There is nothing we could write in this configuration that would allow the
> result to vary based on whether `aws_vpc.example.id` is currently known or not.

> ```
> # There is no "is_known" function like this in the Terraform language
> cidr_block = is_known(aws_vpc.example.id) ? "10.1.5.0/24" : each.value
> ```

> **P9 (A) — Honest Unknown.** Terraform never substitutes a guess for a value it
> does not know. An unknown propagates through every operation that consumes it,
> and the result is unknown unless the result is genuinely determined regardless
> of the unknown input.

> **P10 (A) — Knownness Is Not Observable.** No expression or configuration
> construct may branch on whether a value is known. Knownness is a property of
> the runtime's representation, not of the user model; the *denotation* of a
> configuration must not depend on which phase evaluates it.

P10 is the premise most likely to be violated inadvertently, because a proposed
feature rarely announces itself as exposing knownness. A construct exposes it if
the configuration's meaning — the objects it declares, the values it produces,
the structure of those values — differs according to whether an input was known.

The scope of P10 is deliberately *denotational*, and the boundary matters. It is
not a violation for Terraform itself to behave differently: the CLI emits a
different diagnostic for an unknown `for_each` than for a known one (§5.3.3),
plan rendering displays `(known after apply)`, and the scheduler may defer work
(§8.3). Those are behaviors of the host, not of the program, and no
configuration can observe or branch on them. What P10 forbids is a *language*
construct through which the configuration itself could tell the difference — an
`is_known` predicate, a function that returns a different type for unknown
inputs, or a block that declares different objects depending on knownness.

Type information survives into the unknown, so type errors are still caught:
writing `aws_vpc.example.id.foo` fails during plan even though `id` is unknown,
because the schema says `id` is a string and strings have no attributes.

One nuance is routinely lost and matters for design: **unknownness is not an
intrinsic property of an attribute.** Whether a value is unknown at plan time is
a function of how much the provider can predict, and providers are permitted to
predict more:

> providers are already empowered to return final values during planning if they
> include the logic necessary to do so […] we'd *benefit* from there being fewer
> situations where providers return unknown values, but it will never be possible
> to eliminate them entirely — some values really are determined by the remote
> system only during the apply step.
>
> — Martin Atkins, `hashicorp/terraform#30937` (2022-05-23)

An ARN that a provider could compute from documented syntax but reports as
unknown is a provider limitation, not a law of nature. This has downstream
consequences, because Terraform must treat a newly-unknown value as *possibly
different* from its prior value — and so an avoidable unknown can cause a
needless update, or a needless replacement if the attribute cannot be updated in
place. Unknown does not mean unchanged; it means *unconstrained*, and
conservatism about an unconstrained value is what produces the change.
Reducing unknownness is therefore a legitimate and valuable provider-side
optimization, and one of the few places where a provider can materially improve
plan quality without any Core change.

### 5.3.2 Monotonicity

The property that ties unknowns to P5 is monotonicity:

> Because the Terraform language draws from functional programming principles,
> Terraform can always re-evaluate the same expression once it's gathered more
> information and know that the result will always be strictly a more complete
> version of what it learned on the previous evaluation.

> **P11 (A) — Monotonic Knowledge.** Re-evaluating an expression with more
> precise inputs yields a result no less precise than before: the set of values
> the later result could take must be a subset of, or equal to, the set the
> earlier result could take. In particular, a value that was known must remain
> known and equal.

Note the "or equal to": more information about an input does not oblige the
result to become more known. An operation may legitimately return an equally
unknown result. What is forbidden is *losing* precision or contradicting it.

This is the formal content of "the plan is binding." It is also the invariant
Core checks at the plan/apply seam, and it is stated normatively in cty's own
compatibility policy:

> any operation that was previously returning an unknown value may return either
> a known value or a *more refined* unknown value in later releases, as long as
> the new result is a subset of the range of the previous result.
>
> — `zclconf/go-cty`, `COMPATIBILITY.md`

### 5.3.3 The Pragmatic Compromise at `for_each`

Because unknowns cannot be observed and cannot be guessed, a value unknown at
plan time cannot be used where Terraform requires a known value at plan time.
The best-known instance is `count` and `for_each`, where the *shape of the
graph* depends on the value.

The reasoning behind rejecting it is explicitly a judgement call rather than a
necessity, and it should be cited as such:

> In today's Terraform language, we treat the declaration of a resource block as
> a funny sort of "side-effect". This doesn't necessarily need to be true:
> Terraform could potentially just report that it plans to create some
> undetermined number of `aws_subnet.example` instances, but we intentionally
> made this an error because we concluded that a plan that can't even tell you
> how many objects will be created is not a particularly useful plan.

And the same passage records that Terraform does *not* apply this rule
uniformly:

> With that said, you can see a different variant of this decision for nested
> `dynamic` blocks or `for` expressions involving unknown values: in that case,
> Terraform *will* allow the number of results to be unknown during planning.
> This was a tradeoff for flexibility at the expense of producing an accurate
> plan.

This inconsistency is real, acknowledged, and instructive: the same premise
(P9) admits two defensible resolutions, and Terraform chose differently in two
places for reasons of utility rather than principle. It is a useful reminder
that not every observable behavior is a premise — some are settled tradeoffs,
and the two must not be confused in either direction.

The position has since moved. Core now contains machinery for *deferring*
rather than rejecting: an `expansionDeferred` mode
(`internal/instances/expansion_mode.go`), a `WildcardKey` instance key rendered
`[*]` (`internal/addrs/instance_key.go`), and a `DeferredTransformer` in the
apply graph builder (`internal/terraform/graph_builder_apply.go`). As of v1.16
this machinery is not reachable from the Terraform CLI plan/apply workflow —
core Terraform still errors on an unknown `for_each` — and exists to serve
Stacks (§15.2). A reader should understand deferral as a designed and partially
built capability, not as current CLI behavior.

### 5.3.4 Unpredictable Functions, and One Acknowledged Wart

Functions that cannot return a stable result — `uuid`, `timestamp`, `bcrypt` —
yield *unknown* during planning rather than a value. The mechanism is worth
stating precisely, because it is not what one might assume: the function bodies
compute normally (`internal/lang/funcs/crypto.go:29-44`,
`datetime.go:13-22`). Instead, Terraform classifies them as `impureFunctions`
and, when the evaluation scope is in `PureOnly` mode, wraps them with
cty's `function.Unpredictable`, which forces an unknown result
(`internal/lang/functions.go:20-23,104-113,187,271,289-290`).

Purity is therefore a property of the *scope*, not of the function, and the same
function is unpredictable during planning and concrete during apply. This is the
general pattern for reconciling a genuinely non-deterministic operation with P5:
do not attempt to make the operation deterministic, and do not let the plan
assert its result — suppress knowledge of it at the point where a promise would
otherwise be made. The result behaves exactly like an attribute of a
not-yet-created object, which is the correct analogy.

The `file` function is the documented exception, and Atkins names it plainly:

> The `file` function is, I think, a historical mistake. It dates back to very
> early Terraform releases before we had a strong conception of what promises the
> Terraform language ought to be allowing for, and by the time we realized it was
> too late to treat it as a true unpredictable function because it would break
> many existing configurations.

Changing a file between plan and apply defeats determinism. Terraform still
*detects* it — the safety check fires — but misattributes it:

> It's unfortunate that Terraform mistakenly blames the provider for this, but
> sadly this is caught by the same safety check that catches a provider failing
> to meet the expected contract and so Terraform has to make a guess at who to
> blame here. The important thing, though, is that Terraform detected it and
> stopped before sending this changed value to any remote API.

Two lessons generalize. First, the safety net catches the violation even when
the violation was introduced by Core's own language, which is the behavior one
wants from an invariant check. Second, blame attribution is a design surface of
its own: a check that cannot identify the responsible component will produce
misleading diagnostics for years, and this is a cost to weigh when a check is
placed at a boundary rather than at a cause.

### 5.4 Refinements: Partial Knowledge Without Guessing

An unconstrained unknown is a blunt instrument, and Terraform often knows
*something* about a value without knowing the value. cty supports
**refinements** — constraints attached to an unknown that shrink its range:
non-nullness, a string prefix, numeric bounds, collection length bounds.

The governing rule is one-directional:

> Refinements always *shrink* the range of an unknown value, and never grow it.
> That makes it valid for some operations to ignore refinements and just treat an
> unknown value as representing any possible value of its type constraint.
>
> — `zclconf/go-cty`, `docs/refinements.md`

> **P12 (D) — Refinement Soundness.** A refinement may only narrow the set of values
> an unknown might take, and must never exclude a value the final result could
> legitimately have. A contradictory refinement is a bug, not a conflict to be
> resolved.

Two consequences of shrink-only follow, and both matter for design. Refinements
are *optional to consume*: any component may ignore them and treat the value as
wholly unknown, which is why adding a new refinement kind is not a breaking
change. And refinements are *safe to discard* — at serialization boundaries, for
instance — because dropping them widens the range, and a superset of the true
range is always sound.

Terraform both produces refinements (`refineNotNull`,
`internal/lang/funcs/refinements.go:1-7`; used by `coalesce`, `index`, and
others, `internal/lang/funcs/collection.go:150,170`) and enforces them:
`AssertValueCompatible` checks the applied value against the planned
placeholder's `Range()` (`internal/plans/objchange/compatible.go:243-255`).

Refinements are the correct instrument to reach for when a design is tempted to
guess. A refinement that says "not null" can let a downstream conditional
resolve without saying *what* the value is. Where a refinement narrows enough
that only one value remains, cty may *collapse* it into a known value — an
unknown list known to have exactly two elements can become a known list of two
unknown elements — which is how better knowledge propagates into graph-shaping
decisions without anyone having guessed anything.

### 5.5 Marks: Sensitive and Ephemeral

cty values can carry **marks**: metadata that travels with a value and is
unioned into the results of any operation that consumes it. Terraform defines
`Sensitive` and `Ephemeral` (`internal/lang/marks/marks.go:101-111`), computes
marked paths from provider schema (`internal/configs/configschema/marks.go:19-129`,
`write_only.go:19-72`), and reattaches marks across operations that must
temporarily strip them
(`internal/terraform/node_resource_abstract_instance.go:825,1179,1292,1838,1992,2067,2863`).

Marks and refinements are easily confused and are formally distinct, in a way
worth stating because it predicts which mechanism a new requirement needs:

> Marks should typically be used for additional information that is independent
> of the specific type and value, such as marking a value as having come from a
> sensitive location. […] In a sense the mark represents the *origin* of the
> value rather than the value itself. Refinements are instead directly part of
> the value.
>
> — `zclconf/go-cty`, `docs/refinements.md`

A mark answers "where did this come from, and what handling does that demand?"
A refinement answers "what could this value be?" Marks propagate naively and
must; refinements do not propagate naively and must not.

Contagion is the point of marks. If a sensitive value is used to build another
value, the result is sensitive, because there is no general way to know the
derivation did not preserve the secret.

> **P13 (A) — Mark Propagation.** Marks flow forward through every derivation. A
> value derived from a marked value carries the mark unless something has
> explicitly and deliberately removed it. Any operation that strips a mark must
> justify it, and the burden is on the stripper.

**Sensitive** is a *disclosure* control, not a secrecy mechanism, and any design
leaning on it must say so plainly. Sensitive values are redacted from CLI output
and plan rendering, but are recorded in state in cleartext
(`docs/language/block/variable.mdx:178-200`; `block/output.mdx:137-160`). A
design that treats `sensitive` as protection from an attacker who can read state
is misusing it.

**Ephemeral** makes the stronger claim, and makes it structurally rather than by
redaction: an ephemeral value exists only in memory during a single phase and
must not be written to state or a plan file
(`internal/lang/marks/marks.go:105-111`;
`docs/language/block/ephemeral.mdx:12-34,325-327`). Enforcement is by
construction: write-only attributes are validated null
(`internal/lang/ephemeral/validate.go:15-40`), stripped to typed nulls before
persistence (`strip.go:12-37`), and the ephemeral mark is removed before planned
state is recorded
(`internal/terraform/node_resource_abstract_instance.go:1179-1184`).

> **P14 (P) — Ephemeral Non-Persistence.** An ephemeral value must not appear in any
> artifact that outlives the phase that produced it. A feature that would cause
> an ephemeral value to be persisted, or that cannot determine whether it would,
> must forbid ephemeral values in that position.

P14 is unusual because it cannot be satisfied by careful implementation alone.
It constrains the design surface directly: any position in the language that
feeds a persisted artifact must either reject ephemeral values statically or be
unable to receive them. Ephemerality must therefore be settled at design time
for every construct that accepts an expression. A construct that accepts
ephemeral values in one phase and persists values in another has no correct
implementation — and note that "phase" here includes phases added later, which
is how a construct can become unimplementable after the fact without anyone
changing it.

**Write-only attributes** are the provider-facing form of the same idea: a
schema flag declaring the value is never persisted
(`internal/configs/configschema/schema.go:88-106`). They accept both ephemeral
and non-ephemeral values and are nulled before storage. Their restrictions are
instructive: a write-only attribute may not appear inside a `NestingSet` block
(`internal/configs/configschema/internal_validate.go:96-105`), because set
element identity is derived from element values, and an attribute nulled before
storage would destroy the correlation on which identity depends (§10.3).

### 5.6 The Value Model Is the Plan's Vocabulary

The preceding sections describe the value model as if it were about expressions.
It is more fundamental than that: it is the vocabulary in which the plan is
written. "Known" is how a plan makes a promise. "Unknown, refined not-null" is
how a plan makes a weaker promise honestly. "Null" is how a plan says an
argument was not set, as distinct from set to nothing. "Sensitive" is how a plan
is rendered without disclosure. "Ephemeral" is how a value participates in a run
without entering the record of it.

A feature that needs a concept the value model cannot express is not a feature
with an implementation problem. It is a feature that cannot be planned.

## 6. The Address Algebra

Every object Terraform manages has an **address**: a structured identifier with a
canonical string form. Addresses are how configuration, state, plan, graph, and
user interface all refer to the same thing, and because they appear in error
messages, state files, plan JSON, and `-target` arguments, their string forms are
public contracts (§16).

### 6.1 Static and Dynamic Addresses Are Different Kinds of Thing

The most important structural fact about the address model is that it is two
parallel systems, not one. `internal/addrs` distinguishes **configuration-level**
addresses from **instance-level** addresses at every level of the hierarchy:
`Module` against `ModuleInstance` (`internal/addrs/module.go`,
`module_instance.go`), and `Resource` against `ResourceInstance`, with absolute
forms `AbsResource` and `AbsResourceInstance` (`internal/addrs/resource.go`).

`Module` is a sequence of call names and serializes as `module.a.module.b`.
`ModuleInstance` is a sequence of name-and-key steps and serializes as
`module.a[0].module.b["x"]`. `Resource` is `{Mode, Type, Name}` and names a
configuration block; `ResourceInstance` adds an instance key and names a thing
that can be bound to a remote object.

> **P15 (A) — Static/Dynamic Separation.** A configuration block and a resource
> instance are different kinds of object with different addresses. Exactly one
> of them may be bound to a remote object, and it is the instance. Any construct
> that conflates them — that accepts a configuration address where an instance is
> meant, or vice versa — is ill-formed.

The separation exists because `count` and `for_each` mean the number of
instances is generally not known when the configuration is read. Everything that
must happen before expansion is resolved — graph construction, provider
association, reference analysis — operates on static addresses; everything that
concerns a specific remote object operates on dynamic ones. Recorded state
dependencies, for instance, are `addrs.ConfigResource` rather than instance
addresses (`internal/states/instance_object_src.go:24-78`), because the
dependency is a property of the configuration, not of a particular instance.

### 6.2 Instance Keys

The instance key algebra is deliberately tiny
(`internal/addrs/instance_key.go`): `NoKey` for a singleton, `IntKey` for
`count` and sequence-style `for_each`, `StringKey` for map-style `for_each`, and
`WildcardKey` for an instance whose key is not yet known, rendered `[*]`.

`WildcardKey` deserves attention because it is the address-space expression of
P9. When expansion cannot be resolved — because `count` or `for_each` is unknown
— Terraform does not invent keys. It records that the expansion is deferred
(`expansionDeferred` in `internal/instances/expansion_mode.go`) and represents
the not-yet-enumerable instances with a wildcard. The value model's refusal to
guess is thereby lifted into the address model: Terraform can talk about
instances it cannot name.

### 6.3 Provider Addresses

Providers have two address kinds, and the distinction is exactly the
encapsulation boundary. A `Provider` is a fully-qualified source address —
`hostname/namespace/type`, defaulting to `registry.terraform.io/hashicorp/<type>`
(`internal/addrs/provider.go`). A `LocalProviderConfig` is a module-local name
and optional alias — `provider.aws.foo` — and an `AbsProviderConfig` is the
absolute form including the module and the fully-qualified provider
(`internal/addrs/provider_config.go`).

One rule in that file is worth calling out because it constrains an entire class
of designs: **provider configurations cannot exist inside module instances**.
Parsing rejects module indexes in provider addresses
(`internal/addrs/provider_config.go`). Provider configurations are associated
with static module paths, not dynamic ones. This is why a module containing its
own `provider` block is incompatible with `count`, `for_each`, and `depends_on`
(`docs/language/modules/develop/providers.mdx:31,263-281`) — not as a policy
choice, but because there would be no address for the resulting configuration.

### 6.4 Referenceable, Targetable, and Unique Keys

Three cross-cutting interfaces organize what addresses can *do*.
`Referenceable` marks addresses that may appear in expressions, and carries an
explicit obligation: every implementation must be covered by the evaluation
scope's context construction (`internal/addrs/referenceable.go`). `Targetable`
marks addresses usable with `-target` and supplies containment logic via
`TargetContains` (`internal/addrs/targetable.go`). `UniqueKey` provides a
comparable proxy so addresses can be used as map and set keys
(`internal/addrs/unique_key.go`).

`Referenceable` is the most consequential of the three for feature design,
because it is the formal statement of P2. To make a new kind of object
participate in ordering, it must be referenceable; to be referenceable, it must
be resolvable in the evaluation scope. A new object that is *not* referenceable
cannot participate in the dependency graph, and a design that gives such an
object ordering requirements has, by construction, placed those requirements
outside the mechanism that enforces ordering.

---

# Part III — The Machinery

Part III describes how the representations of Part II are turned into action.
Everything here is downstream: the graph is correct because references are the
sole ordering mechanism (P2), the plan binds because knowledge is monotonic
(P11), and state is meaningful because addresses are stable identities (P15).

## 7. The Dependency Graph

### 7.1 Construction by Transformation

Terraform does not build a graph directly. It builds one by applying an ordered
sequence of **transformers**, each of which adds, removes, or rewires vertices
and edges (`internal/terraform/graph_builder.go`). Each operation has its own
builder because the graphs differ in kind: the plan graph is built from the
configuration, while the apply graph is built from the *changes in the plan*.

The plan pipeline runs roughly thirty transforms
(`internal/terraform/graph_builder_plan.go`). The shape of the sequence matters
more than the individual entries. It begins by introducing nodes from
configuration (`ConfigTransformer`) and from state
(`StateTransformer`, `OrphanResourceInstanceTransformer`); attaches information
to them (`AttachStateTransformer`, `AttachResourceConfigTransformer`,
`AttachSchemaTransformer`); resolves providers (`transformProviders`); then
derives edges (`ReferenceTransformer`, `AttachDependenciesTransformer`,
`DestroyEdgeTransformer`); then prunes and constrains
(`pruneUnusedNodesTransformer`, `TargetsTransformer`); then closes and reduces
(`CloseProviderTransformer`, `CloseRootModuleTransformer`,
`TransitiveReductionTransformer`). The apply pipeline
(`graph_builder_apply.go`) substitutes `DiffTransformer` for the
configuration-derived resource nodes and adds `CBDEdgeTransformer`.

Two structural observations follow, and both constrain feature design.

First, **edge derivation happens at a specific point, after attachment and
before pruning**. A feature that needs an ordering relationship must supply it
as something the reference machinery can see, at that point in the pipeline.
This is P2 made concrete: `ReferenceTransformer` is the mechanism, and anything
outside it is invisible to reduction, to targeting, to destroy-edge derivation,
and to cycle detection.

Second, **orphans come from state, not configuration**. A resource instance in
state with no corresponding configuration block is still a graph node, because
Terraform must plan to destroy it. The graph is therefore never a function of
the configuration alone; it is a function of configuration *and* prior state.

### 7.2 What an Edge Means

An edge means "must happen after" — or equivalently, the graph records
dependencies and Terraform derives ordering by traversing them. Core's own
documentation is explicit that the edges are stored in the dependency direction
rather than the execution direction:

> edges represent dependencies rather than order of operations
>
> — `docs/destroying.md`

This is the pivot on which destroy correctness turns (§12), and it is why the
representation was chosen: reversing execution order is then a property of how
the graph is walked rather than a second graph that must be kept consistent with
the first.

`TransitiveReductionTransformer` removes edges implied by other paths. This is
an optimization of the walk, not a change in meaning: reachability is preserved
exactly. It is worth knowing about because it means the edge set in a built
graph is smaller than the reference set, and any code that inspects edges
expecting to find every reference will not find them.

### 7.3 The Plan Is Not a Graph

A frequent and consequential misconception is that Terraform's plan is a graph
of operations. It is not. The plan is a set of changes — one per resource
instance — without dependency edges. The apply graph is *rebuilt* from those
changes, from the configuration, and from state.

This was considered and not adopted. An early revision of the *Everything is a
Plan* proposal argued that "a graph of operations is a natural representation of
a plan" and criticized discarding dependency information at serialization time;
the argument was dropped from later revisions and never implemented. A reader
should treat plan-as-graph as a live unimplemented idea, not as a description of
Terraform.

The implementation is explicit that a plan is a summary rather than a
self-contained program: a plan "must always be accompanied by the
configuration" it was produced from (`internal/plans/plan.go:18-23`), and the
serialized form carries changes, drift, and deferred items but no edges
(`internal/plans/planfile/tfplan.go:39-150`). `ApplyGraphBuilder` rebuilds the
graph from configuration, changes, and state, with `DiffTransformer` creating
instance nodes from the recorded changes and connecting them to configuration
nodes (`internal/terraform/graph_builder_apply.go:17-24,112-154,213-241`).

The practical consequence is that **the plan file does not carry ordering**, and
so ordering must be re-derivable at apply time from the same inputs. Any feature
whose ordering cannot be reconstructed at apply from configuration, state, and
the recorded changes will not survive the plan/apply boundary. Recorded
dependencies in state (`internal/states/instance_object_src.go:24-78`) exist
partly to make that reconstruction possible for objects whose configuration has
since disappeared.

### 7.4 Walking

The walk is concurrent. Each vertex gets a goroutine, and a vertex executes once
all of its dependencies have completed (`internal/dag/walk.go`). Parallelism is
bounded by a semaphore, ten by default, adjustable with `-parallelism`.

Two behaviors in the walker are worth naming because they define what an error
*means*. Diagnostics are collected per vertex rather than aborting the walk, and
`upstreamFailed` suppresses redundant downstream diagnostics when a dependency
has already failed — so a single root-cause failure produces one error rather
than a cascade. `AlwaysRunVertex` opts out, for nodes that must run regardless
(such as those that close providers).

Nodes may also expand *during* the walk. A node implementing
`GraphNodeDynamicExpandable` produces a subgraph at execution time, which
Terraform validates and walks recursively (`internal/terraform/graph.go`). This
is the mechanism by which a construct whose instance count is not known at build
time can still be executed — and it is the bridge to §8.

## 8. Instance Expansion

### 8.1 Why Expansion Is a Separate Problem

`count` and `for_each` mean the number of objects a configuration block
describes is generally not known when the graph is built. Terraform resolves
this with a two-phase model coordinated by `instances.Expander`
(`internal/instances/expander.go`).

In the first phase, each module call and resource registers its *repetition
mode*: `SetModuleSingle`, `SetModuleCount`, `SetModuleForEach`, and the
corresponding resource variants, plus unknown forms
(`SetModuleCountUnknown`, `SetModuleForEachUnknown`). In the second phase,
instances are enumerated: `ExpandModule`, `ExpandAbsModuleCall`,
`ExpandModuleResource`, `ExpandAbsResource`. The expansion modes are
`expansionSingle`, `expansionCount`, `expansionForEach`, and
`expansionDeferred` (`internal/instances/expansion_mode.go`).

The expander is explicitly order-sensitive: it must be populated in dependency
order, and violating that ordering panics rather than producing a wrong answer.
That choice is itself a statement — an expansion computed out of order is not
recoverable, so the system refuses to continue rather than silently produce an
instance set that does not correspond to the configuration.

### 8.2 Composition and Its Cost

Module expansion and resource expansion compose multiplicatively. The expander
assumes every instance of a module contains the same static objects, differing
only in repetition, which is what makes the composition tractable — but it also
means the instance count of a resource inside a module is the product of the
module's expansion and its own.

This is the concrete reason that P15's static/dynamic separation is not
pedantry. Dependency analysis is performed on *static* addresses precisely
because performing it per-instance would be combinatorial. A feature that
requires instance-level dependency analysis is asking for something the system
deliberately does not do.

### 8.3 Unknown Expansion

When repetition is unknown, Terraform has two possible responses: refuse, or
defer. As of v1.16 the CLI refuses, for the reason quoted in §5.3.3 — a plan
that cannot enumerate what it will create was judged not to be a useful plan.

The deferral machinery nevertheless exists in Core: `expansionDeferred`,
`WildcardKey` rendered `[*]`, and `DeferredTransformer` in the apply builder. It
is not reachable from the CLI plan/apply workflow and serves Stacks (§15.2).

What deferral actually proposes is worth stating precisely, because it is a
change to the *plan model* rather than merely a relaxed validation. A plan today
has two components — the changes it proposes and the state it was derived from.
Deferral adds a third:

> I propose we extend Terraform's plan model with a third bucket: *deferred*
> changes, which describe situations where Terraform knows that there's something
> to do but does not yet have enough information to propose concrete actions.
>
> — *Unknown Values Without Failing* proposal

The third bucket cannot share the representation of the first, and the reason is
a direct consequence of §6.1:

> We cannot just reuse the same diff structure we use for the other two
> components because that relies on us having a complete, known set of resource
> instance addresses, whereas one of the two big reasons for deferral is that we
> don't yet know exactly what instances are desired for a particular resource.

A deferred change is therefore "approximate descriptions of the configuration of
entire resources" — a statement about a *resource* rather than about its
instances, which is precisely what the static/dynamic address split (P15) makes
expressible.

Two properties of this design deserve emphasis for anyone reasoning about
similar problems.

First, **deferral does not violate P9 or P10.** A deferred expansion does not
guess how many instances exist, and it gives the configuration no way to observe
that the count is unknown. It changes what the plan *reports*, not what the
program can *see*. This is the distinction that makes deferral admissible where
an `is_known` predicate would not be.

Second, **deferral does weaken P5, deliberately and visibly.** An ordinary
planned change promises that apply will do this thing to these instances. A
deferred change promises far less — the proposal defines deferred changes largely
by what they lack, and offers no positive guarantee beyond replacing a hard
error. That weakening is why it is gated rather than default:

> Because Terraform today always produces a complete plan or returns an error
> when it cannot, and because automation wrappers such as Terraform Cloud tend to
> rely on that and assume that each configuration changeset maps to one
> infrastructure change, I expect that we'd need to make Terraform CLI initially
> still consider the presence of deferred changes to be a fatal error during
> planning, but we could consider adding a new planning option to tell Terraform
> CLI that it's okay to produce a partial plan

This is a model worth imitating. A feature that weakens a foundational guarantee
is not thereby forbidden — but it must weaken it *explicitly*, in a
separately-identifiable part of the artifact, behind an opt-in, with the
downstream consumers who relied on the strong guarantee named in advance. The
failure mode to avoid is not weakening a guarantee; it is weakening one
silently, so that consumers continue to assume the old promise.

## 9. The Resource Instance Lifecycle

### 9.1 The Object Behind the Address

A resource instance address is bound to at most one **current** object and any
number of **deposed** objects (`internal/states/resource.go:59-68`). The current
object is the remote object the instance presently represents. Deposed objects
are previous remote objects that have been superseded but not yet destroyed —
the state that exists in the window created by create-before-destroy. Deposed
keys are random and unique within an instance
(`internal/states/resource.go:137-178`).

A **tainted** object is one whose creation failed partway through. The status
comment is unambiguous: it marks an object "in an unrecoverable bad state due to
a partial failure," and "a tainted object must be replaced"
(`internal/states/instance_object.go:80-92`). Taint is thus not a user
preference but a record that the object's correspondence to its configuration is
unknown, and the only sound response is replacement.

### 9.2 Replace and Its Two Lowerings

`Replace` is a meta-action. Core lowers it into an ordered pair
(`docs/planning-behaviors.md`):

**Delete then Create** destroys the existing object, then creates a new one at
the same address. This is the default and is the simpler ordering, but it
implies an outage window and it fails when the remote system forbids destroying
an object that something else still references.

**Create then Delete** marks the existing object deposed, creates the new
current object, and then destroys the deposed one. This is what
`create_before_destroy` selects.

Note the structure: Core never selects `Replace` on its own. Something else
selects it — a provider reporting that an attribute cannot be updated in place,
a `-replace` option, a `replace_triggered_by`, or a tainted object — and
`create_before_destroy` then determines which lowering is used. This is the
hybrid pattern described in §2.2, and it is the reason `create_before_destroy`
alone has no observable effect.

### 9.3 Create-Before-Destroy Propagates, and Must

The most important and least intuitive property of `create_before_destroy` is
that it cannot be a local setting. If A depends on B and A is
create-before-destroy, then B must also be create-before-destroy, because the
new A must be created before the old A is destroyed, and the new A requires B —
so B's replacement must also precede the destruction. Terraform therefore
*forces* the flag onto dependencies: `ForcedCBDTransformer` propagates it
upstream, and `CBDEdgeTransformer` then inverts the relevant edges
(`internal/terraform/transform_destroy_cbd.go`).

The consequences are documented as hazards
(`docs/destroying.md`): propagation is mandatory, dependent updates may need to
occur between the creation of a replacement and the destruction of the deposed
object, and a user cannot freely override inherited create-before-destroy
without reintroducing cycles.

This is a clean example of a premise generating obligations. Because ordering
comes only from references (P2), and because create-before-destroy inverts an
ordering, the inversion must propagate along exactly the reference structure
that produced the ordering. There is no local version of this behavior that is
correct.

Atkins has since judged the placement itself a mistake — "with the benefit of
hindsight its placement as a single-resource setting is unfortunate" — and
separately that `prevent_destroy` "should've called this option
`prevent_replace`." Both are naming and scoping errors preserved by
compatibility, and both are useful cautions: a flag whose effect propagates
should not be presented as a property of one resource, and a flag should be
named for what it prevents rather than for the mechanism it intercepts.

## 10. The Core/Provider Contract

### 10.1 Core Does Not Know What Anything Means

Terraform Core has no knowledge of what an EC2 instance is, what it costs, how
long it takes to create, or what it means for one to depend on another. It knows
addresses, values, types, schemas, and a closed vocabulary of actions (P4). All
domain meaning lives in providers.

> **P16 (A) — Core Is Domain-Agnostic.** Terraform Core's behavior must be definable
> without reference to any particular infrastructure domain. Any feature that
> requires Core to understand what a resource *is* has misplaced the logic; the
> knowledge belongs in the provider, reached through the schema-mediated
> protocol.

P16 is what makes the provider ecosystem possible, and it is the premise most
often strained by feature requests that would be easy if only Core knew one more
thing. The correct response is almost always to find the provider-driven
formulation (§2.2).

### 10.2 The Interaction Sequence

The lifecycle of a managed resource instance is a sequence of RPCs
(`docs/resource-instance-change-lifecycle.md:108-363`; protocol in
`docs/plugin-protocol/tfplugin6.proto`):

`ValidateResourceConfig` receives configuration only and returns diagnostics
only. It must tolerate unknown values, because Core may call it early and
repeatedly.

`UpgradeResourceState` converts state written against an older schema version
into the current shape. It may not introduce unknowns — state is a record of
what exists, and nothing about an existing object is unknowable in principle.

`ReadResource` refreshes: it reports the remote object's current condition. It
carries a subtle obligation discussed in §11.3.

`PlanResourceChange` receives prior state, configuration, and Core's *proposed
new state*, and returns a planned new state. Core calls it **twice** per run:
once during planning, possibly with unknowns present, and again during apply
with wholly-known configuration. The consistency between those two calls is
checked.

`ApplyResourceChange` makes reality match the final plan and returns a fully
known new state.

`ImportResourceState` produces a stub state that Core then passes through
`ReadResource`. `MoveResourceState` moves state, private data, and identity
across resource types or providers.

### 10.3 Proposed New State

Before asking a provider to plan, Core computes a **proposed new object** by
merging prior state with configuration
(`internal/plans/objchange/objchange.go:15-505`). The merge rules encode the
semantics of the schema flags:

A non-computed attribute takes its value from configuration. A computed
attribute whose configuration value is null takes the prior value — this is what
makes provider-assigned values stable across runs. `Optional+Computed` is the
muddy case: if the prior value appears to have come from configuration, a null
configuration value stays null rather than reverting to prior
(`:342-380`).

Blocks are correlated by nesting mode: `List` by index, `Map` by key, `Single`
and `Group` by recursion. `Set` is the hard case, and the code says so:

> The correlation for blocks backed by sets is a heuristic…
>
> — `internal/plans/objchange/objchange.go:15-25`

Sets have no element identity. Two set elements are the same element if and only
if they are equal, so an element that is *changing* is indistinguishable from
one element being removed and another added. Core therefore correlates
heuristically, asking for each prior element whether it could plausibly have
been produced by a given configuration element — `validPriorFromConfig`
(`:426-456`) tests whether the prior value is a valid elaboration of the config
value, with computed attributes filled in. This works in ordinary cases and
misbehaves when elements are largely computed.

This is not a defect awaiting repair; it is a consequence of choosing a
value-identity collection. It is the reason write-only attributes are banned
inside `NestingSet` (§5.5), and it is a standing argument for preferring
identified collections when designing a schema.

### 10.4 What Core Asserts

Core validates provider responses at two points, and these checks are the
mechanical expression of P5.

**`AssertPlanValid`** (`internal/plans/objchange/plan_valid.go:13-259`) checks
the provider's plan against the configuration. A provider may not plan absence
where configuration wants existence, nor invent a value for a non-computed
attribute whose configuration value is null (`:34-41`, `:169-178`). It may
substitute a prior value for a configured one only where the two are
functionally equivalent. Write-only attributes must be null in the plan
(`:155-162`). Block counts must match configuration for `List` and `Map`
nestings; for `Set`, Core largely trusts the provider because it cannot
correlate elements (`:180-256`). Nested block elements may not themselves be
unknown — unknownness belongs on attribute values, not on the existence of a
block, because the existence of a block is graph-shaping information. Violations
surface as **"Provider produced invalid plan."**

**`AssertObjectCompatible`** (`internal/plans/objchange/compatible.go:14-265`)
checks the applied result against the plan. Known planned values must equal
actual values. Unknown planned values must be satisfied by an actual value
within the placeholder's range, including its refinements (`:174-205`,
`:243-255`). Null-ness must not flip (`:31-39`). Elements of lists, maps, and
tuples must not appear or vanish (`:205-241`); set element counts may shrink
through deduplication but not grow (`:243-260`). Sensitive values are compared
without disclosing their contents (`:44-61`). Violations surface as **"Provider
produced inconsistent result after apply."**

These two messages are among the most-seen diagnostics in the Terraform
ecosystem, and their status is frequently misunderstood. They are not warnings
about an unusual situation. They are Core detecting that the plan it showed the
operator was not honored, and refusing to proceed as though it had been. The
alternative — applying silently — would make every plan advisory.

### 10.5 Purity Extends to Providers

When providers gained the ability to contribute functions, the purity
requirement had to be made explicit and enforced rather than assumed, because
the extension point is open to third parties:

> Terraform must be able to detect and reject function behavior that isn't
> "pure", because Terraform's plan/apply model relies on expressions always
> producing the same result during apply as they did during plan unless the
> expression result was explicitly an unknown value.
>
> — *Functions in Providers* proposal

> The function must always return an identical result given the same set of
> arguments, and must have no observable side-effects. That is, it must behave
> as a pure function. (Technically Terraform Core can only verify the pure
> function behavior, and even then only to a limited extent. However, a provider
> that violates this rule is incorrect even if Terraform doesn't catch it.)

The parenthetical is the general principle for every contract in this document,
and it deserves to be lifted out: **a violation is a violation whether or not
Core detects it.** Detection is a courtesy. Correctness is defined by the
contract.

The same proposal illustrates the namespace hazard that recurs throughout
Terraform's design. Provider functions were given their own namespace
(`provider::name::function`) rather than being added to the built-in namespace,
because:

> We have been bitten in the past by making namespaces that contain mixture of
> both built-in names and externally-defined names.

Terraform has three such mixed namespaces already — resource type names against
predefined symbols like `var` and `path`; resource arguments against
meta-arguments like `count` and `lifecycle`; and provider configuration
arguments against their meta-arguments — and each one means that adding a
reserved word is potentially a breaking change. Any new extension point should
assume a separate namespace unless there is a strong reason otherwise.

### 10.6 The Legacy SDK Exemption

Protocol versions 5 and 6 coexist, with parallel implementations in
`internal/plugin` and `internal/plugin6`. Within them is a compatibility
concession that every Core engineer eventually meets: the `legacy_type_system`
flag, which allows the original helper/schema SDK to violate rules that would
otherwise be errors. The protocol comment is blunt about its intended audience:

> ==== DO NOT USE THIS ==== … in all other SDKS
>
> — `docs/plugin-protocol/tfplugin6.proto:319-350`

Related machinery includes `NormalizeObjectFromLegacySDK`
(`internal/plans/objchange/normalize_obj.go:11-27`), which reshapes null and
unknown nested blocks into forms the legacy SDK expects, and which the file
itself notes is incompatible with computed blocks under protocol v6 and must not
be used there. `AssertNoLegacyBehavior`
(`internal/configs/configschema/schema.go:49-73`) exists to determine whether a
schema can be held to modern semantics.

The lesson is about the shape of the debt rather than its details. An exemption
granted at a contract boundary does not stay at that boundary. It propagates
into the merge algorithm, the plan validator, the normalization layer, and the
schema validator, and every subsequent feature must be defined twice — once for
conforming providers and once for exempt ones. This is the mechanism by which a
compatibility concession becomes a permanent tax on design velocity.

## 11. State

### 11.1 State Is a Binding, Not a Cache

Terraform state is frequently described as a cache of remote object attributes.
That description is wrong in the way that matters. State's essential content is
the **binding** between an address and a remote object: this configuration
block, at this instance key, corresponds to *that* object in the remote system.
The attribute values are secondary — they support diffing and are refreshable —
but the binding is not derivable from anywhere else.

> **P17 (A) — One Address, One Object.** Within a single state, a remote object
> is bound to exactly one resource instance address, and a resource instance
> address is bound to at most one current remote object. Terraform's guarantees
> about an object hold only for objects it owns through such a binding.

The scoping to "within a single state" is not a weakening but an accurate
statement of what Terraform can enforce. Nothing prevents two independent
configurations from each binding the same remote object; Terraform has no
cross-state registry and cannot detect it. That is precisely the failure `data`
blocks are designed to make unlikely (§1.4), and it is why the ownership
question is a *modelling* discipline imposed on practitioners rather than an
invariant Core can check.

> **P23 (A) — State Is Wholly Known.** A state snapshot contains no unknown
> values. Unknownness is a property of a *plan*, which describes a future; state
> describes what exists.

P23 concerns the *representation*, not completeness of information: state may
legitimately omit real facts about a remote object — write-only attributes are
nulled before storage (§5.5), and no provider records everything the remote
system knows. What it may not do is record a value as unknown. Every value
present in state is concrete.

P23 is the sharpest asymmetry in the system and is easy to miss because nothing
announces it. It is the reason `ReadResource` and `UpgradeResourceState` may not
return unknowns (§10.2), and it is a hard constraint on any feature that would
like to record a not-yet-determined value:

> Currently we don't allow `ReadResource` to return unknown values *at all*,
> because there is a deep assumption in Terraform that a state snapshot is always
> wholly-known.
>
> — Martin Atkins, `hashicorp/terraform#30937` (2022-07-20)

Relaxing it is not a local change. It would require a representation for an
*unknown prior value* in plan rendering — the same comment sketches
`example = (unknown) -> "Hello"` — and would propagate into every consumer of
state, including third-party ones (§11.2).

P17 explains a cluster of otherwise unrelated facts. It is why Terraform cannot
simply query the remote API instead of keeping state: the API can enumerate
objects but cannot say which configuration block is responsible for which
object. It is why importing requires an explicit address. It is why `data`
blocks assert existence rather than returning absence (§1.4) — the failure being
prevented is two configurations each concluding they own the same object. And it
is why the refactoring constructs of §14 are *rebinding* operations that do not
touch the remote system.

P17 is also why a "disabled resource" has nowhere to live. An instance key is
what distinguishes an instance from its resource whenever there is not exactly
one; a conditionally-absent resource has zero or one instances and no key to
distinguish the case, which means no address, which means no binding
(`hashicorp/terraform#21953`, 2022-08-16). The address algebra is not a
convenience layered over state — it *is* how state identifies things.

The state model reflects this: `State` → `Module` → `Resource` →
`ResourceInstance` → object, with per-object metadata comprising recorded
`Dependencies` (as *configuration* addresses),
`CreateBeforeDestroy`, `Private` provider data, `SchemaVersion`, and — more
recently — `IdentitySchemaVersion` and `IdentityJSON`
(`internal/states/instance_object_src.go:24-78`).

### 11.2 State Is an Artifact Contract

The state file is read by tools Terraform does not control. Its format is
versioned (v1 through v4 readable, `internal/states/statefile/read.go:43-47`),
and unsupported versions produce an explicit diagnostic rather than a
misparse (`:125-190`).

Two fields carry more meaning than their names suggest. `Serial` increments on
every modification and is how conflicting updates are detected. `Lineage` is
assigned once when a state is created and *never updated*, and exists so that
Terraform can tell whether two serial numbers are even comparable — two states
with different lineages are unrelated histories, and comparing their serials
would be meaningless (`internal/states/statefile/file.go:17-31`).

Lineage is a good model for identity design generally: it is opaque, compared
only for equality, and carries no parseable structure, which means no external
workflow can come to depend on its internals.

### 11.3 Refresh and Drift

Refresh asks each provider what its objects currently look like, and the
answers are compared against prior state to detect **drift** — change that
occurred outside Terraform.

The `ReadResource` contract contains a subtlety that is a frequent source of
provider bugs: the provider should return the *prior value* where the remote
system has merely normalized a value, and the *remote value* where the object
genuinely changed (`docs/resource-instance-change-lifecycle.md:252-293`). The
provider is being asked to distinguish a cosmetic difference from a real one,
because Core cannot. Getting this wrong in one direction produces perpetual diff
(a P6 violation); getting it wrong in the other hides real drift.

Refresh became part of planning rather than a separate state-mutating step in
v0.15.4, and `-refresh-only` was introduced as a planning *mode*
(`docs/cli/commands/plan.mdx:96-135`). The standalone `terraform refresh`
command is deprecated precisely because it wrote to state without review; the
documentation now directs users to `terraform apply -refresh-only` so that
detected drift is presented for approval before being committed
(`docs/cli/commands/refresh.mdx:29-54`). This is P3 reclaiming a historical
exception: an operation that used to have unreviewed side effects was converted
into a plan.

Drift is reported through `Context.driftedResources`
(`internal/terraform/context_plan.go:1132-1239`), which reports out-of-band
deletions as `Delete`, changed values as `Update`, and moved objects as `NoOp`
with a changed previous address. The plan carries two distinct prior states to
make this expressible at all: `PriorState`, the refreshed view of the remote
system, and `PrevRunState`, the state as recorded at the end of the previous run
(`internal/plans/plan.go:46-61`). Drift is precisely the difference between
them, which is why it can be reported without being confused with a planned
change.

One consequence is worth noting because it is easy to get wrong when adding a
new kind of object: even when refresh produces no planned change, the refreshed
values are still written into the working state, because otherwise "any output
values referring to this will not react to the drift"
(`internal/terraform/node_resource_plan_instance.go`). Detecting drift and
*propagating* it are separate obligations, and a feature that does the first
without the second will silently produce stale downstream values.

### 11.4 Resource Identity

Resource identity is a newer mechanism — a structured, provider-defined,
object-typed key stored alongside state
(`IdentitySchemaVersion`/`IdentityJSON`), with protocol support via
`GetResourceIdentitySchemas` and `UpgradeResourceIdentity`, and a language
surface in `import { identity = { … } }`
(`docs/language/block/import.mdx:65-100`).

It exists because the traditional `id` attribute was a single opaque string
being asked to serve as a universal primary key. Many remote systems identify
objects by a composite — region plus name, account plus path, parent plus child
— and encoding that into one string forced every provider to invent its own
undocumented separator convention, which then became a de facto contract with
users who had to type those strings into `terraform import`. Identity replaces a
string-encoding convention with a typed structure, which is the same move as
replacing address parsing with an address algebra (§6).

## 12. Destroy as Reversal

### 12.1 The Ordering Is Derived, Not Declared

Destroy ordering is the reverse of create ordering: if B depends on A, then B is
destroyed before A. This is why edges are stored as dependencies rather than as
execution order (§7.2) — reversal is then a property of traversal.

> **P18 (D) — Destroy Is Reversal.** Destroy order is derived by reversing the same
> dependency structure that determines create order. There is no separate
> destroy-ordering mechanism, and any ordering fact that is not in the dependency
> structure is unavailable during destroy.

P18 is a sharp instrument in design review because it converts a vague worry
into a specific question: *what does this feature's ordering look like
reversed?* A feature that has a coherent answer during create and no answer
during destroy is incomplete, and the incompleteness will not be visible in any
create-path test.

The reversal is not limited to `terraform destroy`. The same inversion applies
within an ordinary plan to any portion of the graph being removed — a resource
deleted from configuration, an instance dropped by a changed `for_each`, a
module removed. Destroy ordering is therefore exercised by routine changes, not
only by whole-configuration teardown.

### 12.2 Destroy Edges and Their Hazards

`DestroyEdgeTransformer` derives destroy ordering from the relationship between
creators and destroyers, using both configured and *recorded* dependencies
(`internal/terraform/transform_destroy_edge.go`). The recorded dependencies
matter because an object being destroyed frequently has no configuration left to
consult — which is exactly the situation §14.3 addresses at the language level.

The code documents genuine hazards. Cross-provider destroy edges can produce
cycles. During a full destroy, provider dependencies can require resources to
remain alive until evaluation completes, because a provider configuration may
itself depend on a resource that is scheduled for destruction. These are not
incidental bugs; they are the friction generated where the reversal meets
objects whose lifetimes are not themselves part of the reversed graph.

---

# Part IV — Composition and Extension

Part IV concerns scale: how a configuration is decomposed, how it is changed
over time, how the Terraform model has been extended into adjacent languages,
and why compatibility is a property of the system's semantics rather than a
release policy layered on top of them.

## 13. Modules

### 13.1 Modules Are Namespaces, Not Runtime Boundaries

A module is a unit of authorship, distribution, and naming. It is not a unit of
execution. Terraform flattens the entire module tree into a single graph, and
resources in different modules are ordered relative to one another by exactly the
same reference mechanism that orders resources within a module.

This is easy to state and easy to forget, and forgetting it produces designs
that assume a module boundary provides isolation it does not provide.

> **P19 (P) — Modules Are Namespaces.** A module boundary constrains *visibility* and
> *naming*, not evaluation or ordering. There is one graph, one evaluation, and
> one plan for the whole configuration. A module does not execute; its contents
> do.

Flattening was a deliberate choice with real benefits: resources in sibling
modules may be created concurrently rather than serialized by module, and a
module's input may legitimately depend on another module's output which itself
depends on the first module's output, because the dependency graph is
per-object rather than per-module. Terraform is unusual in permitting this, and
real modules depend on it.

It has also proven to be a constraint. Because there is no module-level
evaluation boundary, there is no natural place to put per-module state,
per-module provider configuration, or per-module planning. Atkins has
characterized the flattening as having "been a significant constraint on a
number of different potential Terraform language features in the past," and
Stacks' `component` deliberately gives it up (§15.2) — a component *is* a
runtime boundary in a way a module is not.

### 13.2 The Module Interface

The interface is narrow by design: input variables in, output values out. A
caller cannot reach into a module to reference a resource directly, and a module
cannot reach outward to reference its caller's objects. `depends_on` on a
`module` block is a coarse instrument that makes every object in the module
depend on the given target, which is the only thing it *can* mean given P19 —
there is no module-level node to attach a finer relationship to.

The `nullable`, `sensitive`, and `ephemeral` arguments on variables, and
`sensitive`/`ephemeral` on outputs, exist because the interface is the right
place to state such constraints: a module author can refuse null at the boundary
rather than defending against it throughout (§5.2), and can declare that a value
crossing the boundary carries handling obligations (§5.5).

### 13.3 Providers Cross the Boundary Differently

Provider configurations are the one thing that does not follow the
variables-in/outputs-out model, and the rules are worth stating precisely
because they are a frequent source of surprise
(`docs/language/modules/develop/providers.mdx`):

Provider configurations are global to the whole configuration. Only the root
module should define them. A child module inherits *default* provider
configurations implicitly, but never inherits aliased ones — those must be
passed explicitly via the `providers` meta-argument.

And the rule with the sharpest consequences: a module that contains its own
`provider` block cannot be used with `count`, `for_each`, or `depends_on`. As
§6.3 establishes, this is not a policy decision. `AbsProviderConfig` addresses
contain a static `Module`, not a `ModuleInstance`, and parsing rejects module
indexes outright. There is no address for "the provider configuration inside the
third instance of this module," so there is no such object.

This is a good example of the address algebra doing load-bearing work. The
restriction looks arbitrary at the language level and is inevitable at the
address level, which is why P15 is worth understanding before proposing
anything that touches provider association.

### 13.4 Custom Conditions Are Assertions, Not Policy

`variable` validation, `precondition`, `postcondition`, and `check` blocks form
a family of author-written assertions. Their design contains a decision worth
generalizing:

> These checks are intentionally attached to objects already in the graph, rather
> than being new graph nodes in their own right, in the hope of making it easier
> for module authors to understand what other parts of a module are guarded by a
> particular condition.
>
> — *Preconditions and Postconditions* proposal

Attaching rather than adding was chosen for *comprehensibility*, and it produces
precise semantics: a precondition gates evaluation of the object's arguments; a
postcondition gates evaluation of everything downstream. The proposal is also
explicit about a boundary that follows from §8:

> A precondition does *not* block evaluation of the `count` or `for_each`
> expression of a resource, but in return for that limitation the condition
> expression may refer to `each.key`, `each.value`, and `count.index`.

Expansion must be resolved before instances exist, so an instance-scoped
condition cannot gate it — and in exchange the condition gets to see the
instance's identity. Conditions are evaluated as soon as their inputs become
known, surfacing at plan time where possible and apply time otherwise, which is
P9 applied to diagnostics.

`check` blocks differ from the other three in the one respect that defines their
purpose: a failed `check` assertion is a **warning** and does not block the
operation, whereas a failed `precondition` or `postcondition` is an **error**
that does (`docs/language/block/check.mdx:14-24`;
`internal/moduletest/run.go:215-233`, which notes that `terraform test`
deliberately promotes check-block warnings back to failures so that tests can
assert on them). That difference makes `check` the right tool for monitoring an
assumption and the wrong tool for enforcing a rule.

None of these constructs are policy: they are written by the module author,
evaluated in the module's own scope, and bypassed by editing the module. Policy
in the governance sense must be evaluated by something the module author does
not control.

## 14. Refactoring: Changing Identity Without Changing Objects

### 14.1 The Problem

P17 binds an address to an object. Refactoring changes addresses. Without a
mechanism to rebind, renaming a resource or moving it into a module would read
to Terraform as the deletion of one object and the creation of another — which
is exactly what it would then do.

`moved`, `removed`, and `import` are the configuration-driven answers. Their
common property is that they operate on *bindings*, not on remote objects. A
`moved` block causes Terraform to rename an entry in state before planning, then
plan as though the object had always been at the new address
(`docs/language/block/moved.mdx:1-46`). No API call is made.

### 14.2 Inertness

The design property that keeps these constructs within the declarative model is
that they are not commands:

> Similar to the existing `moved` blocks, this design captures some details about
> the way the module configuration has changed over time. On its own a `removed`
> block is inert: it's a statement about history, not a direct imperative
> command for Terraform to act on.
>
> — *`removed` blocks* proposal

An inert statement takes effect only if the situation it describes is found in
prior state. Running the same configuration twice is therefore safe: the second
run finds nothing to rebind and does nothing. This is what distinguishes these
blocks from `terraform state mv` and `terraform state rm`, their imperative
predecessors — the imperative forms execute unconditionally, are not reviewable
as part of a plan, are not shared with collaborators through version control,
and are not repeatable.

> **P20 (P) — Refactoring Statements Are Inert.** A construct that records a change
> in configuration over time must describe a condition and its resolution, not an
> action. It must be a no-op when the condition does not hold, so that applying
> the same configuration repeatedly is safe.

Terraform enforces the coherence of these statements strictly
(`internal/refactoring/move_validate.go`): a move to the same address is an
error, a `from` address that still exists in configuration is an error, one
source may have only one destination and one destination only one source, and
cycles are errors. Moves are executed by topologically sorting a graph of move
statements (`move_execute.go`), so chains and nested moves reduce to a single
canonical outcome.

### 14.3 Metadata Outlives State

The `removed` block exists because of a genuine hole in the desired-state model,
and the proposal's diagnosis is the most valuable general principle in the
corpus on this subject:

> A general way to frame the above concerns is by considering the idea of
> different categories of information having different "lifetimes": when making
> changes to the remote system, the desired state is outlived by the current
> state of the remote system. The *metadata* about objects, which is separate
> from either desired or current state, MUST outlive both the desired state and
> the current state of the remote system.
>
> — *`removed` blocks* proposal

In a pure desired-state language, deleting a block means "destroy this." But the
block also carried the information needed to *perform* the destruction — which
provider configuration to use, what it depends on, its destroy-time
provisioners and connection settings. Deleting the block deletes the means of
carrying out the instruction it implies.

> **P21 (P) — Metadata Lifetime.** Information required to *remove* an object must
> outlive both the desired state that declared it and the current state of the
> object itself. A design that stores removal-relevant metadata only in the
> declaration has made the declaration undeletable in practice.

The `removed` block is also instructive for what it *excludes*: `count`,
`for_each`, `prevent_destroy`, `ignore_changes`, `replace_triggered_by`, and
`postcondition` are all forbidden, each for a stated reason. A postcondition is
meaningless because no object will remain to evaluate it against, so a `removed`
block's only guarantee to downstream objects is non-existence. `prevent_destroy`
is incoherent because destroying and forgetting are the only available actions
and forgetting is already expressed by `destroy = false`. Individual instances
cannot be addressed, because the block describes removal of the whole
configuration block rather than of its dynamic instances — P15 again.

This exclusion list is a model for how to specify a new construct. Each
exclusion is derived from what the construct *means*, not from implementation
convenience.

## 15. The Derived Language Family

Terraform now anchors a family of HCL-based languages. They share a substrate:
HCL parsing and decoding, the cty value model and marks, the `addrs` address
model, provider schemas and clients, and — to varying degrees — the plan and
state machinery. Understanding which parts are shared and which are forked is
the practical content of "designing a feature that cooperates with everything
else."

### 15.1 Terraform Test

Test files (`.tftest.hcl`) contain `run` blocks, each of which executes a `plan`
or `apply` against the module under test and evaluates `assert` blocks
(`internal/configs/test_file.go:98-124`;
`docs/language/tests/index.mdx:31-36`). Mocking is provided by `mock_provider`,
`override_resource`, `override_data`, and `override_module`
(`docs/language/tests/mocking.mdx:27-31,124-190`).

Test is not a separate runtime. It drives the ordinary module runtime through
`moduletest.TestSuiteRunner` (`internal/command/test.go:74-163`). This is its
most important architectural property and the reason it is a credible testing
mechanism: a `run` block exercises the same code path a user does.

It also means Test is a *consumer* of every other feature. A language feature
that has no representation in the Test runtime is a feature that modules using
it cannot test. This is the most common form of the cooperation failure the
source paper warns about, because Test integration is easy to defer and its
absence is invisible until an adopter has already committed.

### 15.2 Stacks

Stacks introduces two file types — `.tfcomponent.hcl` for what infrastructure
exists and `.tfdeploy.hcl` for where and how many times it is deployed
(`docs/language/files/stack.mdx:13-66`) — with `component`, `provider`,
`variable`, `output`, `removed`, and `locals` on the component side, and
`deployment`, `deployment_group`, `identity_token`, `store`, `upstream_input`,
and `publish_output` on the deployment side.

Its relationship to Terraform is explicit:

> an orchestration layer on top of zero or more trees of Terraform modules
>
> — `internal/stacks/README.md:1-13`

Planning a component constructs a `terraform.Context` and calls
`tfCtx.Plan(moduleTree, state, opts)`
(`internal/stacks/stackruntime/internal/stackeval/planning.go:108-171`).
`stackconfig`, `stackplan`, and `stackstate` are Stacks' analogues of `configs`,
`plans`, and `states`.

Two divergences matter foundationally. First, a `component` is a runtime
boundary in a way a module is not (§13.1): it has its own plan and its own
state, which is precisely the isolation module flattening denies. Second, Stacks
handles unknown `for_each` by deferring rather than erroring
(`internal/stacks/stackruntime/internal/stackeval/for_each.go`), which is the
resolution of the §5.3.3 tension — and the reason the deferral machinery exists
in Core.

Stacks is also where the compatibility argument has been renegotiated. Because
adopting Stacks is voluntary and opt-in, it can make changes that would be
impossible in the Terraform language proper. This turns out to be the mechanism
by which Terraform has evolved past its compatibility promises: not the
"language editions" mechanism designed for the purpose (§16.4), but adjacent
opt-in languages.

### 15.3 Query

Query files (`.tfquery.hcl`) contain `list` blocks that enumerate existing
remote objects, with `provider`, `count`/`for_each`, `include_resource`,
`limit`, and a nested `config` block
(`internal/configs/query_file.go:27-154`). Providers implement the listing.
Query's output can be `-generate-config-out` — generated configuration for
discovered objects (`internal/command/query.go:33-43`).

Query is a CLI command over the normal machinery, running as
`backendrun.OperationTypePlan` with `Query = true` (`:115-170`). Its `list`
abstraction is deliberately distinct from `data` sources: `data` asserts
existence of one known object (§1.4), whereas `list` discovers an unknown
population. Conflating them would have broken the assertion semantics that make
`data` blocks useful.

### 15.4 Actions

An `action` block declares a provider-defined operation, with a nested `config`
block and `count`/`for_each`/`provider` meta-arguments
(`internal/configs/action.go`; `docs/language/block/action.mdx:50-125`). Actions
are invoked either directly (`terraform plan -invoke=…` /
`apply -invoke=…`) or by an `action_trigger` block nested in a resource's
`lifecycle`, on one of six events: `before_create`, `after_create`,
`before_update`, `after_update`, `before_destroy`, `after_destroy`
(`internal/configs/action.go:48-92`). A special `caller` symbol is available in
action configuration when invoked from a trigger. Actions do not affect resource
state (`docs/language/invoke-actions.mdx:12-16`).

Actions sit at the intersection of most of this document's premises and are
therefore the best available exercise for a reader testing their understanding.

The load-bearing question is P2. An `action_trigger` establishes ordering by
naming a *position in time* — `before_create`, `after_destroy` — relative to the
resource whose `lifecycle` block contains it. Under the test in §1.2, the
question to ask is not whether data flows between the action and the resource,
but whether the required ordering is expressible as a reference from one address
to another. A trigger's relationship to its host resource plausibly is: the
trigger occurs inside the resource's own block and names actions in the current
module, so there is an address-to-address relationship available. What is less
obvious, and what a design review should insist be answered explicitly, is
whether the resulting ordering constraint becomes a real edge in the single
dependency graph — and therefore participates in cycle detection, transitive
reduction, targeting, and destroy-time reversal — or whether it is sequencing
applied *within* a node at execution time. Ordering of the second kind satisfies
no premise in this document, because none of the machinery that reasons about
ordering can see it.

The remaining questions follow the checklist in Appendix B. Destroy-time
triggers must be coherent under reversal (P18), and `before_destroy` in
particular orders an action against an object that is ceasing to exist. Because
actions are invoked at points where different values are available, the
ephemerality question (P14) must be answered separately for each of the six
events rather than once for the construct. Because actions do not affect state,
their effects fall outside P6's convergence guarantee — a triggered action is
not idempotent in the way a resource change is — and outside P24's plan
authorization unless the plan represents the invocation explicitly. And because
`caller` introduces a symbol whose meaning depends on invocation context, it is
worth checking against P8.

None of these observations is a verdict. They are the questions the premises
generate, and the point of the exercise is that they are generated mechanically
rather than discovered by someone who happens to remember that destroy reverses
the graph.

### 15.5 Policy

In v1.16 policy is not a public HCL language in the sense the others are. What
exists is an evaluation subsystem: a policy client with setup and evaluation
requests (`internal/policy/policy.go:1-140`), wired into `init`, `plan`,
`apply`, and `query`, with `.tfpolicy.hcl` appearing in tests and diagnostics.
Results attach to diagnostics and to query-row metadata. It should be described
as a hosted-policy subsystem under development rather than as a settled member
of the family.

### 15.6 What the Family Reveals

The family shares a substrate but has not converged on a vocabulary. Its units
of execution are `run`, `list`, `component` plus `deployment`, and `action` plus
`action_trigger`. Its repetition semantics differ: Test runs are sequential or
parallel, Query and Actions use `count`/`for_each`, and Stacks has both
`for_each` on components and replication at the deployment layer. Its handling
of unknown values differs most of all — Stacks defers where Terraform errors.

None of this is necessarily wrong; different problems warrant different
surfaces. But it is a standing cost. Each divergence is a thing a practitioner
must learn separately and a thing every future cross-cutting feature must
implement several times. The generalization worth carrying into design review is
that **a divergence in surface is cheap to introduce and expensive to
maintain**, and that the cost is paid by people who work across more than one of
these languages — which, increasingly, is everyone.

## 16. Compatibility as a Semantic Property

### 16.1 The Promise

From v1.0, HashiCorp committed that modules written for v1.0 would continue to
plan and apply without changes throughout the v1.x series; that automation built
around a defined workflow subset would keep working; and that providers built
against the documented protocol would remain compatible without recompilation
(`docs/language/v1-compatibility-promises.mdx`).

The promise covers the language's top-level blocks and meta-arguments, its
operators and built-in functions, the provider wire protocol, and the provider
and module installation protocols. It explicitly does *not* cover natural-language
CLI output or log text — "not a stable interface" — while JSON output modes and
exit codes *are* the supported machine interfaces.

Two scoping rules in that document deserve emphasis because they are routinely
misread. First, **the promises apply only to valid configurations**: "We consider
a configuration to be valid if Terraform can create and apply a plan for it
without reporting any errors." A configuration that currently errors may error
differently later. Second, error detection may move *earlier*: "A configuration
that generates errors during the apply phase might generate similar errors at an
earlier phase in future, because we generally consider it better to detect
errors in as early a phase as possible." Moving a diagnostic earlier is
explicitly permitted; moving one later, or turning a valid configuration into an
error, is not.

### 16.2 Why This Is Semantic, Not Procedural

Compatibility is usually discussed as release discipline — a thing enforced by
review and changelog. In a language it is stronger than that, because the
artifacts are durable and the dependants are unknown.

> **P22 (P) — Expressible Means Permanent.** Within the compatibility window and
> across the supported interfaces, anything the language permits a user to
> express will be relied upon. For valid configurations, a construct's syntax,
> semantics, and machine-readable artifacts are contracts, whether or not they
> were designed as such.

The qualifications are load-bearing and are drawn from §16.1 rather than added
here: the promise covers *valid* configurations, the *supported* interfaces
(JSON output and exit codes, not prose diagnostics or log text), and the *v1.x
window* rather than eternity. A diagnostic may be reworded, and may be moved to
an earlier phase. What may not happen is that a configuration which planned and
applied cleanly stops doing so.

The consequence for design is the one the source paper draws: the decisive
question about a new construct is not only what it enables but what it
*permits*. Every degree of freedom left open becomes a contract, and every future
enhancement must interoperate with everything that freedom allowed. Narrowing
later is not available; the promise is precisely that it is not.

This is the strongest available argument for closed designs. A closed design
moves complexity from the user to the language, and — more importantly — leaves
room to open up later, which is a move that remains available in a way that
closing down does not.

### 16.3 The Namespace Hazard

Terraform's compatibility exposure is unusually high for a structural reason
identified in the *Language Editions* proposal:

> The Terraform language is particularly susceptible to new features causing
> breaking changes because the fundamental design of the language has several
> situations where unrelated namespaces overlap with one another

The overlaps are: resource type names against predefined symbols like `var` and
`path`; resource and provider arguments against meta-arguments like `count` and
`lifecycle`; and the same for provider configuration.

> In all three of these cases, a namespace of reserved words defined as part of
> the language coexists with terms defined by an ecosystem extension point.
> Therefore adding new reserved words in those contexts always risks breaking any
> external artifact that was already using that name.

Every new meta-argument is therefore potentially a breaking change against some
provider's existing attribute name. The `enabled` proposal (§1.4) foundered
partly on exactly this: adding an `enabled` meta-argument to `resource`, `data`,
and `module` blocks is "a compatibility hazard for any existing resource type
which has an argument named `enabled` or any module which has a
`variable "enabled"` declaration" (`hashicorp/terraform#21953`, 2022-08-16).
Note the shape of that argument — the feature was not rejected on its merits but
on a namespace collision that is invisible unless one is looking for it.

This is why provider functions were given a separate namespace (§10.5), and it
is the single most transferable lesson in this section: **new extension points
should get new namespaces.**

### 16.4 Editions: Designed, Dormant

The *Language Editions* proposal designed a mechanism for evolving past the
promises — versioned editions of the language, selected per module, so the
ecosystem could adopt changes gradually rather than at a cliff edge:

> so that we can later release new iterations of the language which may not be
> entirely compatible with previous iterations, while still retaining support for
> the older iterations and — crucially — allowing the modules in a configuration
> to differ in which language iterations they use

It has not shipped. The `language` argument exists as a stub, and `TF2021`
remains the only edition. A reader should treat editions as a
designed-but-dormant capability, not an operating feature — and should note that
the pressure it was designed to relieve has instead been relieved by opt-in
adjacent languages (§15.2).

### 16.5 Stated Principles Are Revisable; Premises Are Less So

It is worth closing with a caution against reading this document, or any design
document, too rigidly. In 2019 Terraform's language redesign concluded with the
expectation that "this will be the last significant shake-up of the Terraform
language for the foreseeable future." It was not: module `for_each`, `moved`
blocks, preconditions and postconditions, `check` blocks, `import` blocks and
configuration generation, provider-defined functions, `removed` blocks,
ephemeral values, write-only attributes, and an entire sibling language have all
followed.

Several specific decisions have been publicly reconsidered by their authors: the
`file` function ("a historical mistake"), `prevent_destroy`'s name ("we should've
called this option `prevent_replace`"), `create_before_destroy`'s placement as a
per-resource setting ("unfortunate"), the treatment of variable absence versus
null ("an unfortunate historical error"), the mixing of built-in and
author-defined argument namespaces, and the splat operator, which "probably
wouldn't have passed our design principles" had it been new.

The distinction this document has tried to maintain throughout is between these
— *decisions*, which are revisable and several of which were wrong — and
*premises*, which are the assumptions the rest of the system is built on. The
former should be argued about freely. The latter should be disturbed only
deliberately, with the burden of proof on the proposer, and with the
understanding that the cost of disturbing one is not contained by the feature
that disturbs it.

---

# Appendices

## Appendix A — Index of Claims

Claims are cited by number in design review. Each is tagged **(A)** axiom,
**(D)** derived guarantee, or **(P)** design policy, per §0.2 — an objection
citing an axiom is much stronger than one citing a policy, and a reviewer should
say which they mean.

The **Requires** column has one meaning: *if the listed claim is relaxed without
replacement, this claim is no longer guaranteed.* It is a support relation, not
a consequence relation. Derived guarantees always require something; axioms
mostly require nothing.

| # | Kind | Name | Statement | § | Requires |
|---|---|---|---|---|---|
| **P1** | A | Data Flow | Configuration expresses relationships between values, not a sequence of operations. | 1.2 | — |
| **P2** | A | Reference Order | Evaluation order is determined solely by references — occurrences of a `Referenceable` address, whether or not traversed to a value. | 1.2 | — |
| **P3** | A | Pure Evaluation | Evaluating a configuration produces values and a description of intent, never side effects. | 2.1 | — |
| **P4** | P | Closed Action Vocabulary | The set of state-transition actions Core proposes for a managed object is fixed; providers influence which, not which exist. | 2.2 | P16 |
| **P5** | D | Value Fidelity | What the plan states as known must hold after apply; what it leaves unknown must resolve within published constraints. | 2.3 | P3, P9, P11, P12 |
| **P24** | D | Plan Authorization | Every externally visible apply effect must have been represented in the reviewed plan — subject, action class, and existence. | 2.3 | P3, P15 |
| **P6** | D | Convergence | A successful, complete apply reaches a fixed point with respect to its configuration. | 3.1 | P5, P24, P17, P23 |
| **P7** | A | Schema-Directed Interpretation | Configuration meaning requires a schema; syntax alone does not determine argument vs. block, or type. | 4.1 | — |
| **P8** | P | One Evaluation Semantics | One expression language with one denotation; restricted contexts restrict what may be *referenced*, not what expressions mean. | 4.3 | P11 |
| **P9** | A | Honest Unknown | Unknown evaluation is a sound over-approximation, never a guess. | 5.3.1 | — |
| **P10** | A | Knownness Is Not Observable | No configuration construct may branch on whether a value is known. | 5.3.1 | — |
| **P11** | A | Monotonic Knowledge | More precise inputs yield a result no less precise; known values remain known and equal. | 5.3.2 | P3, P10 |
| **P12** | D | Refinement Soundness | A refinement only narrows an unknown's range and never excludes a legitimate final value. | 5.4 | P9, P11 |
| **P13** | A | Mark Propagation | Marks flow forward through every derivation; stripping one requires justification. | 5.5 | — |
| **P14** | P | Ephemeral Non-Persistence | An ephemeral value must not appear in any artifact outliving the phase that produced it. | 5.5 | P13 |
| **P15** | A | Static/Dynamic Separation | Configuration blocks and resource instances are different objects with different addresses; only instances bind to remote objects. | 6.1 | — |
| **P16** | A | Core Is Domain-Agnostic | Core's behavior must be definable without reference to any infrastructure domain. | 10.1 | P7 |
| **P17** | A | One Address, One Object | Within a single state, a remote object binds to exactly one instance address and an instance address to at most one current object. | 11.1 | P15 |
| **P23** | A | State Is Wholly Known | A state snapshot contains no unknown values; unknownness belongs to plans. | 11.1 | — |
| **P18** | D | Destroy Is Reversal | Destroy order is the reversal of the dependency structure that determines create order. | 12.1 | P2, P21 |
| **P19** | P | Modules Are Namespaces | A module boundary constrains visibility and naming, not evaluation or ordering. | 13.1 | — |
| **P20** | P | Refactoring Statements Are Inert | Constructs recording configuration change describe a condition, not an action, and are no-ops when it does not hold. | 14.2 | P3, P17 |
| **P21** | P | Metadata Lifetime | Information required to remove an object must outlive the desired state that declared it and the object itself. | 14.3 | P17 |
| **P22** | P | Expressible Means Permanent | Within the compatibility window and across supported interfaces, anything expressible for a valid configuration is a contract. | 16.2 | — |

**Reading the table.** P18 requires P21 as well as P2, because reversal needs the
dependency structure to survive the deletion of the configuration that declared
it — which is what recorded state dependencies and `removed` blocks provide.
P6 requires P23 because a fixed point cannot be computed against a prior state
that is not concrete. P5 requires P12 because once refinements form part of a
plan's published constraint, unsound refinements make the plan's promise false.

**Numbering.** P23 and P24 are numbered out of layer order because they were
identified after the others; P23 belongs with the state claims (§11) and P24
with the plan claims (§2.3). Numbers are stable citations and are therefore not
reassigned.

**A caution on use.** Naming a claim starts a conversation; it does not end one.
Several of these — P4, P14, P17, P21, P22 — have scope questions this document
does not fully settle (how small is "small"? which artifacts? for how long?).
Where a dispute turns on such a boundary, the honest move is to say that the
boundary is undefined and to define it as part of the proposal, rather than to
assert the claim as though its edges were sharp.

## Appendix B — Interaction Checklist for Feature Design

The purpose of this checklist is not completeness — it cannot be complete — but
to make the interaction surface *enumerable*, so that "I did not think about
that" becomes a finding rather than a discovery made after implementation.

**Ordering and the graph.** Does the feature create any ordering requirement? Is
that requirement expressed as a reference (P2)? If it is expressed some other
way, stop and redesign. Is the object referenceable, and is it covered by the
evaluation scope? What does the ordering look like *reversed* (P18)? Does it
behave correctly when its subject is being destroyed, when it is being replaced,
and when it is being removed from configuration by a `for_each` change rather
than by editing?

**Values.** What happens when an input is unknown? Does the feature behave
differently depending on knownness — including by producing a different
diagnostic, a different structure, or a different number of objects (P10)? Can it
accept a sensitive value, and if so where can that value end up (P13)? Can it
accept an ephemeral value, and does every phase in which it operates have a
non-persisting path (P14)? Does it distinguish null from empty (§5.2)?

**The plan.** Is the effect representable in the plan? Can an operator see it
before it happens (P3)? Is it checkable at apply — that is, can Core tell whether
what happened matches what was planned (P5)? Does the feature's effect survive
the plan/apply boundary given that the plan carries no dependency edges (§7.3)?

**Expansion and addressing.** Does the feature apply to a configuration block or
to an instance (P15)? What is its address, and what is the string form of that
address — remembering that the string form is a contract (P22)? Does it compose
with `count`, `for_each`, and module expansion? What is its behavior when
expansion is unknown?

**State.** Does the feature create, consume, or invalidate a binding (P17)? Does
it add anything to state, and if so is the state format change readable by older
versions and writable in a way third-party consumers will tolerate (§11.2)? What
metadata does removal require, and does that metadata survive the deletion of
the declaration (P21)?

**Providers.** Does the feature require Core to understand something about a
resource's domain (P16)? Does it require a protocol change, and if so does it
work with protocol v5 as well as v6? What does a legacy-SDK provider do with it
(§10.6)? What happens if a provider implements it incorrectly — is that
detectable, and will the resulting diagnostic name the right component (§5.3.4)?

**Composition with modules.** Does it work inside a child module? Does it work
when that module is expanded? Does it need to cross a module boundary, and if so
does it cross via variables and outputs or by some other route (P19)?

**The language family.** Does it work under `terraform test` — and if not, have
you accepted that modules using it are untestable (§15.1)? Does it have a
meaning under Stacks, where components are runtime boundaries and unknown
expansion defers rather than errors (§15.2)? Is it reachable from Query,
Actions, or policy evaluation, and should it be?

**Surfaces and environments.** Is it expressible in JSON as well as native
syntax (§4.2)? Does it work in the JSON plan output and the machine-readable UI?
Does it work locally, in CI, in HCP Terraform and Terraform Enterprise, through
agents, and in an airgapped environment?

**Compatibility.** Does it add a name to any namespace that also contains
ecosystem-defined names (§16.3)? Does it constrain anything that previously
validated? Does it change an existing diagnostic's text, level, or phase — and
if the phase moved, did it move earlier (permitted) or later (not) (§16.1)? What
new contracts does it create, and is their value worth honoring them
indefinitely (P22)?

## Appendix C — Exception Register

A premise index is only usable alongside an honest list of the places the system
does not satisfy it. Each entry below names what is relaxed, whether the
relaxation is quarantined, and what it costs. An argument of the form "we
already do X" should be checked against this table: most of these are debt, and
two of them are the reason particular subsystems are hard to change.

| Exception | Relaxes | Quarantined? | Notes |
|---|---|---|---|
| **Provisioners** | P1, P3, P24 | Partly — opt-in per resource, documented as last resort | Configuration-authored imperative steps whose effects are not represented in the plan and whose results are not recorded in state. Their pathologies (unplannable, non-idempotent, no rollback, taint-on-failure as the only recovery) are consequences of the relaxation, not independent defects. |
| **`terraform state mv` / `rm` / `import` (CLI)** | P3, P24 | No | Unconditional, unreviewed state mutation. Superseded for most uses by `moved`, `removed`, and `import` blocks (§14), which are the P20-conforming replacements. |
| **`terraform taint` / `untaint`** | P3, P24 | No | Legacy; superseded by `-replace`, which is a plan option and therefore reviewable. |
| **`terraform refresh` (standalone)** | P3, P24 | No — deprecated | Wrote refreshed state without review. Replaced by `-refresh-only` planning mode, an instance of a historical exception being successfully converted into a plan (§11.3). |
| **`-target`** | P6, P24 | Yes — explicit operator opt-in per run | Produces a state that is by construction not a fixed point of the full configuration. Documented as a recovery tool, not a workflow. |
| **Legacy SDK (`legacy_type_system`)** | P5, and the `AssertPlanValid` rules | Yes — per-provider flag | The most expensive entry in this table. Because the exemption sits at a contract boundary, it propagates into the merge algorithm, plan validator, normalization layer, and schema validator, and every subsequent feature must be defined twice (§10.6). |
| **The `file` function** | P11 | No | Reads mutable external state during evaluation; changing the file between plan and apply defeats determinism. Detected but misattributed to the provider (§5.3.4). Retained for compatibility; acknowledged as a mistake. |
| **`dynamic` blocks and `for` expressions with unknown inputs** | P24 | No | May produce an unknown *number* of results, where `count`/`for_each` refuse. An acknowledged inconsistency, chosen for flexibility at the cost of plan accuracy (§5.3.3). |
| **Deferred changes** | P24 | Yes — separate plan bucket, gated, not CLI-reachable | The model instance of how to weaken a guarantee correctly: explicit, separately represented, opt-in, with affected consumers named in advance (§8.3). |
| **Actions** | Open — see §15.4 | Partly | Whether trigger-derived ordering produces real graph edges determines whether P2 is preserved. Effects are outside P6 and, unless invocations are represented in the plan, outside P24. |

Two entries deserve to be read together. The legacy SDK exemption and deferred
changes both weaken a foundational guarantee, and the difference between them is
entirely in the *manner*. Deferral is a named bucket, opt-in, with a stated list
of consumers whose assumptions it breaks. The legacy exemption is a boolean on a
wire protocol that silently changes what Core will tolerate. One of them is
tractable to remove; the other has been load-bearing for a decade.

## Appendix D — Known Gaps in This Document

The following are within the document's stated scope but are not adequately
treated, and are recorded here so that their absence is not mistaken for their
unimportance.

**Concurrency and state integrity.** `Serial` and `Lineage` are described
(§11.2), but the invariant that prevents two concurrent runs from acting on the
same binding — locking, atomic state update, and stale-plan rejection — is not
stated as a premise. It almost certainly should be.

**Partial failure and durable progress.** §3.2 excludes incomplete applies from
P6 but says nothing about what *must* remain true after an apply fails partway.
This is the foundation on which recovery workflows rest, and it is missing.

**Provisioner semantics.** Provisioners appear only in Appendix C and as
removal metadata in §14.3. Because they are configuration-authored imperative
operations, a full treatment would say precisely how their invocation is
disclosed and how they relate to P1, P3, P5, P24, and P6.

**Premise applicability across the derived languages.** §0.4 states that P1–P24
govern Terraform configuration and Core, and §15 notes specific divergences, but
there is no systematic matrix showing which premises each derived language
preserves, revises, or declares inapplicable. Stacks in particular revises at
least P19 and P24.

**Provider-side premises.** This document is written from Core's perspective.
The obligations a provider takes on — determinism, normalization-versus-drift
discrimination (§11.3), identity stability — deserve a parallel treatment that
does not exist here.

## Appendix E — Source Map

**Core design documents in `hashicorp/terraform`.** `docs/architecture.md` is the
canonical subsystem overview. `docs/planning-behaviors.md` states the default
planning behavior and the three-pattern taxonomy of special behaviors.
`docs/resource-instance-change-lifecycle.md` is the normative Core/provider
contract. `docs/destroying.md` covers destroy ordering and
create-before-destroy. `docs/plugin-protocol/` holds the protocol buffers
definitions for protocol versions 5 and 6.

**Implementation.** `internal/addrs` — the address algebra. `internal/configs`
and `internal/configs/configschema` — configuration model and schema.
`internal/lang`, `internal/lang/marks`, `internal/lang/ephemeral` — expression
evaluation and marks. `internal/plans/objchange` — proposed-new-object
computation and the plan/apply assertions. `internal/terraform` — graph
builders, transforms, and node implementations. `internal/dag` — the graph and
its walker. `internal/instances` — the expander. `internal/states` and
`internal/states/statefile` — the state model and its serialization.
`internal/refactoring` — `moved`, `removed`, and `import` statements.
`internal/providers`, `internal/plugin`, `internal/plugin6` — the provider
interface and protocol clients. `internal/stacks`, `internal/moduletest`,
`internal/policy` — the derived languages.

**External normative sources.** `zclconf/go-cty`: `COMPATIBILITY.md` states the
"unknown values can become more known" rule; `docs/refinements.md` specifies
refinements and their shrink-only discipline; `docs/marks.md` specifies marks.
`hashicorp/hcl` defines the structural and expression syntax.

**Public documentation.** `docs/language/v1-compatibility-promises.mdx` is the
normative statement of the compatibility promises. Note that
`docs/internals/graph.mdx` is substantially out of date — it still describes
"interpolations" and resource meta-nodes from the pre-0.12 architecture — and
should not be relied upon; prefer `docs/architecture.md` and the implementation.

**Essays by Martin Atkins (`apparentlymart`), at log.martinatkins.me.**
*Evolving the Terraform Language* (2019-03-01) on the v0.12 design principles.
*Terraform is a Data Flow Language* (2019-08-19) on the data-flow paradigm.
*Unknown Values: The Secret to Terraform Plan* (2021-06-14) — the single most
important source on unknown values, and the origin of the unknowns-are-not-
promises argument.

**Proposals in `hashicorp/terraform-proposals`.** *Everything is a Plan*;
*Preconditions and Postconditions*; *Functions in Providers* (the clearest
statement of the purity contract); *`removed` blocks* (the metadata-lifetime
principle); *Language Editions* (the namespace-overlap analysis). Note that
authorship varies and should be verified with `git log --follow` before
attributing a quotation to an individual; several documents are written in team
voice. Note also the chronology: *Functions in Providers* postdates *Language
Editions* and *applies* its namespace diagnosis, rather than restating it — the
two make related but distinct arguments and should not be conflated.

**On verifiability.** Citations in this document fall into three tiers, and
readers should treat them differently. *Public and durable*: the Terraform
repository, its `docs/` directory, the public documentation set, `go-cty`, and
the Atkins essays — all independently checkable, and safe to cite outward.
*Public but volatile*: GitHub issue and pull request comments, which are
checkable but can be edited and are not authored as specifications. *Internal*:
`terraform-proposals`, which is not visible outside HashiCorp/IBM and which
several readers of this document will be unable to verify. Where a claim rests
only on the third tier, it is doing so because no public statement of it exists;
that is a signal the idea is under-documented, and where such an argument
matters it should be restated from a public source if one can be found.

**A caution on provenance.** Several load-bearing ideas in this document are
recorded only in essays, proposals, and issue comments rather than in
documentation or code comments. Where a premise is stated here but its only
citation is a blog post or a proposal, that is a signal that the premise is
under-documented in the codebase, not that it is weakly held. P2 (Reference
Order) and P23 (State Is Wholly Known) are the two clearest instances: both are
absolutely load-bearing, and neither is written down anywhere a new Core
engineer would naturally encounter it.
