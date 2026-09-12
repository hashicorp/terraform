# The Foundations of Terraform

### Language semantics, runtime contracts, and architectural dependencies

**Status:** Draft  
**Audience:** Terraform Core engineers, architects, and language designers  
**Classification:** Internal

---

## Abstract

Terraform combines a data-flow configuration language with a runtime that plans
and applies changes to external systems. Its behavior depends on several
representations working together: references establish evaluation dependencies;
typed values carry partial knowledge; addresses distinguish declarations from
instances; and state records the objects Terraform manages. Providers connect
these domain-independent mechanisms to particular remote systems.

This paper describes those foundations in four layers: the programming model,
the configuration and value representations, the execution machinery, and the
mechanisms for composition and evolution. It distinguishes expression evaluation
from provider operations, a plan's value commitments from its authorized changes,
and configuration structure from the instances created from it. Numbered
principles and an architectural dependency table provide a reference for design
discussions. They are an account of the system's assumptions and contracts, not
a formal proof of the implementation.

The purpose is to support reasoning across feature boundaries. A proposed
extension must be understood in terms of the assumptions it uses and changes,
including consequences for planning, state, composition, and compatibility.

## 0. Scope and Conventions

### 0.1 Architectural Reasoning

A Terraform feature rarely belongs to only one subsystem. An expression may
determine instance keys; those keys become addresses in a plan and state;
references to those addresses establish evaluation dependencies; and saved
dependency information helps order destruction after the configuration changes.
Local behavior therefore needs to be understood in the context of the whole
system.

The central design principle of this paper is that **an objection identifying a
conflict with an established architectural premise requires an answer, even when
the reviewer cannot predict a particular future failure**. The reviewer should
identify the premise and explain where the proposal conflicts with it. The
design response can show that the premise is preserved, replace it together
with the contracts that depend on it, or define a separate contract with a clear
boundary. A changed guarantee must be visible to its consumers.

This is a method for examining designs, not a claim that Terraform's present
architecture is immutable. Nor does a conflict establish that a particular
failure is inevitable. It establishes work that the design must account for.
The discussion of interacting language features in the supplied
*Hitchhiker's Guide to Language Design* motivates this approach.

### 0.2 Three Kinds of Claim

The numbered principles use three labels:

| Label | Meaning in this paper |
|---|---|
| **Axiom (A)** | An architectural assumption on which the present model is built. |
| **Derived guarantee (D)** | A property supported by those assumptions and by the implementation contracts described alongside it. |
| **Design policy (P)** | A deliberate choice about the language or its supported behavior. |

These labels are explanatory, not a formal classification of the source code.
An axiom here is not a mathematical axiom, and a derived guarantee is not a
theorem proved solely from the other numbered statements. Guarantees have
conditions: convergence, for example, requires stable inputs and a provider that
can implement the desired result.

The labels also do not rank the importance of commitments. Ephemeral
non-persistence is classified as a policy because it is a chosen language
contract; violating it is not less serious for that reason. Any proposed change
must consider the behavior promised to users and other components, whatever
label this paper assigns it.

Appendix A records architectural dependencies. Its entries identify assumptions
that help explain or support a principle. They are neither exhaustive nor
necessary-and-sufficient conditions, and removing one assumption does not prove
that every possible implementation of the dependent property is impossible.

### 0.3 Organization

**Part I** describes data flow, planning, and convergence. **Part II** covers
configuration, values, and addresses. **Part III** follows those representations
through graph construction, expansion, resource operations, providers, state,
and destruction. **Part IV** covers modules, refactoring, related languages, and
compatibility.

The order is explanatory rather than a strict implementation dependency order.
The appendices provide the principle index, a design checklist, scope notes,
coverage limits, and a source map.

### 0.4 Evidence and Terminology

Implementation references use the `hashicorp/terraform` checkout at
[`05cecbb315e87abc2c2bdb182963dd8e36d574fb`][core], dated 11 September 2026.
Documentation references use the supplied Terraform **v1.16.x (RC)** and Plugin
Framework **v1.18.x** trees. These are distinct source baselines: a capability in
the implementation is not necessarily available in the documented CLI release.
Version-sensitive behavior is identified where relevant.

Source links identify files at that revision, and symbol names locate the
relevant implementation within them. Public documentation links are reading
aids; the supplied versioned trees are the documentation baseline.
Essays and issue discussions explain rationale. Proposals describe possible
designs unless adoption is established separately. None of these source types
should be substituted for another.

**Core** means Terraform's evaluation and planning runtime and its supporting
packages, as distinct from the CLI command layer, providers, and surrounding
automation. **Expression evaluation** means computation of a value from an
expression and its evaluation context. A **graph walk** is a broader operation:
it can evaluate expressions, call providers, and update working state.

The principles primarily describe the Terraform configuration language and Core.
Related languages may use Core while adding their own evaluation or
orchestration rules (§15). Their contracts must be considered separately.

---

# Part I - Programming Model

## 1. Data Flow and References

### 1.1 Declarative Programming

Terraform is a programming language in the declarative tradition. A
configuration describes desired objects and their relationships rather than an
imperative procedure for constructing them. "Declarative" is useful at that
level, but it is too broad to settle most language-design questions.

The more specific description is **data flow**. Expressions describe how values
are obtained and combined, while references establish dependencies that Core
uses to determine evaluation order. Martin Atkins develops this distinction in
[*Evolving the Terraform Language*][evolving-language] and
[*Terraform is a Data Flow Language*][data-flow].

### 1.2 Evaluation Order

Conditional expressions select values. `for` expressions construct values.
`dynamic` blocks construct nested configuration, and resource or module
`for_each` selects a set of instances. These constructs can affect the work
Terraform needs to do, but they do not give the author an imperative sequence
of execution steps.

> **P1 (A) - Data Flow.** A Terraform configuration describes objects, values,
> and their relationships. Core derives an evaluation order from those
> relationships rather than executing the configuration as an ordered program.

> **P2 (A) - Reference Order.** Evaluation order is determined solely by
> references. A dependency requiring one configuration object to be evaluated
> before another must be expressed through a reference.

A reference is not itself a value. In `aws_instance.example.id`, the reference
identifies `aws_instance.example`; the remaining traversal selects information
from its value. In `depends_on = [aws_instance.example]`, the reference expresses
a dependency without contributing an argument value. `depends_on` is therefore
an application of P2, not an exception to it.

References must be resolvable before full evaluation of the objects they relate.
An expression cannot make an otherwise invalid reference valid merely by
putting it in a branch that happens not to be selected. James Bardin explains
the connection between static reference analysis and graph construction in
[Terraform issue #21953][static-references].

P2 concerns **evaluation order**, not every sequencing constraint in a Terraform
operation. The execution graph also represents provider setup and shutdown,
instance expansion, and the ordering of create and destroy operations.
Lifecycle transformations and saved dependencies participate in that machinery
(§7, §12), adding execution constraints beyond the references in expressions.

### 1.3 A Useful Analogy

A spreadsheet illustrates the distinction between specifying a dependency and
choosing an execution order. A formula names the cells it uses; the calculation
engine schedules work and detects cycles. Formula position is not an instruction
to execute cells in that order.

Terraform adds concerns that a spreadsheet does not have: persistent object
identity, provider operations, partial knowledge, and an approval boundary before
managed-resource changes. The analogy explains the evaluation model, not the
whole runtime. Other data-flow systems discussed in Atkins' essay are useful for
the same limited purpose. [Source: [*Terraform is a Data Flow Language*][data-flow].]

### 1.4 Declarations and Conditional Instances

A declaration and its instances are different things. With `count = 0` or an
empty resource `for_each`, the declaration remains available to static analysis
but denotes an empty collection of instances. Expressions can inspect that
collection or derive other collections from it without making the declaration
itself conditionally referenceable.

This representation explains why repetition is more than syntactic convenience.
It gives absence a value-level representation while preserving a stable
configuration namespace. A proposal for conditional declarations would need to
define reference resolution and the value of an absent declaration explicitly;
it cannot simply inherit the existing semantics of an empty instance collection.
The discussion in [#21953][conditional-resources] explores this distinction.

Existence checks against remote systems are a separate concern. Some data
sources require a particular object to exist and report its absence as an
error; others return collections, empty results, or computed information.
Those behaviors belong to the provider's data-source contract. Reading an
object and deciding which configuration owns its lifecycle are not equivalent
operations (§11).

### 1.5 Language-Design Preferences

The v0.12 design work articulated three preferences: prioritize readers over
writers, prefer explicit behavior, and keep simple tasks simple while allowing
complex ones. They explain choices in expression syntax and language
regularity, but they do not decide every tradeoff by themselves.

For example, familiar shorthand can remain useful even when a more explicit
form exists. The relevant question is whether the shorthand has a coherent
meaning and composes with the rest of the language, not whether every construct
maximizes one preference in isolation. [Source:
[*Evolving the Terraform Language*][evolving-language].]

## 2. Planning and Applying

### 2.1 Expressions and Operations

Terraform separates computing values from carrying out managed-resource
changes. Expressions contribute desired values; Core compares configuration
with prior and refreshed state; providers help determine the proposed changes.
Apply then attempts those changes under the plan's constraints.

> **P3 (A) - Pure Evaluation.** Expression computation is distinct from
> resource operations. Evaluating an expression is not an imperative request
> to perform a managed-resource change; those changes belong to the runtime's
> planning and application protocol.

This scope is important. Planning is not free of external interactions: it can
refresh managed resources, read data sources, and open ephemeral resources.
Initialization and state operations have their own effects. Some functions also
read external inputs or produce unpredictable results, which require specific
handling (§5.3.4). P3 separates these operations from expression computation;
it does not restrict all external interaction to apply.

Apply also evaluates configuration. It does not execute a source-independent
script containing every final value. As previously unknown inputs become known,
Core evaluates dependent configuration and checks that the resulting values
remain consistent with the plan.

The design guidance in [`docs/planning-behaviors.md`][planning] favors the
plan/apply process for externally visible changes while acknowledging existing
operations outside that process.

### 2.2 Core's Action Model

For ordinary managed-resource changes, Core works with a small action model:
creation, update, deletion, replacement, and no-op. Reads account for data-source
operations. Replacement includes an ordering choice, normally delete-then-create
or create-then-delete.

The implementation also represents other operations, including forgetting a
binding and the open/renew/close lifecycle of ephemeral resources. The familiar
CRUD list is therefore an introduction, not an exhaustive enumeration of
`plans.Action`. [Sources: [planning behaviors][planning];
[`internal/plans/action.go`][plan-actions].]

> **P4 (P) - Closed Action Vocabulary.** Core defines the action classes used
> by its planning and execution protocols. Providers implement and influence
> those operations; they do not independently add action classes to Core's
> resource-change model.

The provider may require replacement for a changed attribute or supply a more
precise planned value. Core decides how the resulting change is represented and
scheduled. Provider-defined capabilities must fit an explicit Core protocol;
they do not acquire lifecycle semantics merely by being exposed by a plugin.

The planning design document distinguishes three common sources of special
behavior:

| Source | Scope | Examples |
|---|---|---|
| Configuration | Follows the module or resource declaration | `ignore_changes`, `create_before_destroy`, `moved` |
| Provider | Reflects the remote system's requirements | Replacement requirements and planning adjustments |
| Run options | Applies to the operator's particular invocation | `-replace`, `-refresh-only`, `-target` |

These categories can overlap. Their value is to make the responsible party and
lifetime of a decision explicit. A run option, for example, may require support
from every UI that wraps Core, whereas a configuration setting travels with the
module. [Source: [planning behaviors][planning].]

### 2.3 What a Plan Commits To

A plan contains both proposed changes and statements about values. These are
related but distinct commitments.

> **P5 (D) - Value Fidelity.** Within the planned change contract, a value
> recorded as known must remain equal when the change is applied. An unknown
> value may become known, but the result must satisfy the constraints recorded
> for that unknown.

> **P24 (D) - Plan Authorization.** Applying a plan is limited to the changes
> that plan authorizes. Resolving unknown values does not authorize additional
> subjects or a different class of change.

P5 is a consistency rule. Core checks planned and actual object values using
the compatibility checks in `internal/plans/objchange`. P24 describes the scope
an operator approves. Those checks support the contract, but cannot establish
that an arbitrary provider implemented every remote API call correctly.
The plan describes Terraform-level changes, not a complete trace of provider
implementation details.

Neither principle promises that apply will succeed. An API may reject a change,
credentials may expire, or a provider may return an inconsistent result. Core
must report the failure rather than silently substitute a different proposal.
A reported failure also does not imply that no remote changes have occurred
(§3.2, §11.4).

Deferral concerns a third property: **completeness**. A partial plan can leave
work for a later planning round without relaxing either its value commitments
or the scope of its executable changes. Deferred work is not authorized for
execution by the current plan; it must be planned subsequently (§8.3).

## 3. Convergence and Its Limits

### 3.1 Stable Results

Terraform aims to converge managed objects on the declared configuration.
Under stable conditions, applying the proposed changes should leave no further
changes to propose for that same desired result.

> **P6 (D) - Convergence.** Given stable configuration, inputs, and relevant
> external conditions, a complete successful apply should leave the managed
> objects consistent with the desired result, so that a subsequent plan
> proposes no further changes to them.

The qualifications are part of the principle. A changing input such as
`timestamp()` can intentionally produce a new desired value on the next run.
A provider may also need to account for API normalization or eventual
consistency before the remote object can be represented stably.

An unexpected recurring difference is useful evidence of a mismatch among
configuration, provider planning, applied results, and refreshed state. It
does not, by itself, identify which component is responsible. The relevant
contracts are described in the [resource-instance lifecycle][resource-lifecycle]
and the provider planning rules (§10).

### 3.2 Scope, Drift, and Partial Failure

Terraform observes external systems during operations, not continuously.
Changes made elsewhere between runs can produce **drift**, even after a
previously convergent apply. Refresh establishes a new observation; it does not
retroactively invalidate the earlier run.

Convergence is also bounded by ownership and scope. A configuration does not
control every object a provider can observe. Targeting, deferral, or an
interrupted apply may leave parts of the configuration unprocessed. Such an
operation does not establish whole-configuration convergence, although the
result may happen to require no further changes.

Finally, apply is not an atomic transaction over all remote systems. Earlier
changes can succeed before a later operation fails. Terraform records the
results it can and reports the errors; recovery normally proceeds from that
updated state rather than through a general rollback (§11.4).

### 3.3 Representational Requirements

These contracts explain the representations that follow. References must be
available for dependency analysis before their values are necessarily known.
Unknown values must preserve what a plan can and cannot promise. Addresses must
distinguish a declaration from the concrete instances named in a change.
State must preserve enough identity and lifecycle information to continue
managing objects across runs and configuration revisions.

---

# Part II - Configuration, Values, and Addresses

## 4. The Configuration Language

### 4.1 Syntax and Schema

Terraform uses HCL for two related purposes: a structural language of bodies,
blocks, and arguments, and an expression language that computes argument values.
Terraform v0.12 brought expression syntax into that common model, replacing the
earlier separation between HCL structure and HIL string interpolation.
[Source: [*Evolving the Terraform Language*][evolving-language].]

Native syntax distinguishes a block from an attribute:

```hcl
example {
  name = "service"
}

example = {
  name = "service"
}
```

The first form is a block named `example`; the second assigns an object
expression to an argument named `example`. Parsing establishes that syntactic
distinction. A schema establishes whether either form is permitted in the
containing body and how its contents are interpreted.

> **P7 (A) - Schema-Directed Interpretation.** Syntax alone does not determine
> the full meaning of a configuration body. Its schema defines the permitted
> arguments and block types, their value constraints, and their nesting rules.

Core defines schemas for its own constructs. Providers supply the schemas for
provider configurations and resource types. Consequently, tools can parse
configuration and perform substantial structural and reference analysis without
a provider schema, but cannot complete provider-specific validation without the
corresponding schema information. Installing providers is the normal CLI route
to obtaining that information; it is not a logical prerequisite for every form
of static analysis.

In `internal/configs/configschema`, a `Block` contains attributes and nested
block types. Attributes carry a type or nested object schema and flags such as
`Required`, `Optional`, `Computed`, `Sensitive`, and `WriteOnly`. Nesting modes
include single, group, list, set, and map. `DecoderSpec()` derives an HCL decoding
specification, `ImpliedType()` derives a cty type, and `CoerceValue()` converts a
value to the shape required by the schema. These are related operations, not
interchangeable descriptions of parsing. [Source: [configuration schemas][schemas].]

### 4.2 Native and JSON Forms

Terraform's JSON configuration syntax is another representation of the same
language. It exists primarily for generators and tools. Its conventions for
blocks and expressions must be interpreted according to the surrounding
Terraform schema; a JSON object alone does not establish whether its contents
represent a Terraform object value or nested configuration.

A language feature therefore needs a defined JSON representation as well as
native syntax. This does not require identical notation or preservation of
comments and formatting. It requires that generated configuration can express
the feature's semantics without relying on a native-only escape mechanism.
[Source: [JSON configuration syntax][json-syntax].]

### 4.3 Restricted Evaluation Contexts

Some expressions must be evaluated before the normal resource graph can run.
Module installation is an example: Core needs to resolve a module's source
before it can load that module's configuration.

In the supplied v1.16 documentation, module `source` and `version` can use
constant expressions involving eligible variables and local values. Input
variables used for this purpose must declare `const = true`. This provides
limited early evaluation without permitting resource results to determine what
configuration must first be installed. [Sources: [module blocks][module-doc];
[input variables][variable-doc].]

> **P8 (P) - One Evaluation Semantics.** Restricted evaluation contexts use
> the same expression language. Their restrictions concern available inputs
> and capabilities; they must not silently give ordinary expressions a
> different meaning.

The contexts are not identical. A resource reference may be unavailable during
initialization, and an unpredictable function may yield an unknown during
planning. A design should specify those restrictions directly. Reusing the same
surface syntax does not, by itself, establish a coherent relationship between
evaluation phases.

## 5. The Value Model

Terraform uses [`cty`][cty] to represent values and types. The model includes
ordinary concrete values, nulls, unknowns, refinements, and marks. These are
used in expression evaluation and in the contracts between Core and providers.

### 5.1 Types and Conversion

The primitive types are `string`, `number`, and `bool`. Lists, sets, and maps
have a common element type. Objects describe named attributes that can have
different types, while tuples describe positional elements that can have
different types.

The distinction matters during conversion. A tuple can be converted to a list
only if its elements can be converted to a common element type. An object can
be converted to an object constraint with fewer attributes, but the omitted
attributes do not remain available through the converted value. Module
interfaces therefore use type constraints to define a view of a value, not
merely to check its name.

`any` in a Terraform type constraint is a placeholder for a type to be inferred,
not a concrete type containing arbitrary unrelated values. For example,
`list(any)` still requires a single element type. At the implementation level,
`cty.DynamicPseudoType` represents the absence of a concrete type constraint;
it should not be confused with a language-level universal type.
[Sources: [type constraints][type-constraints]; cty's `convert` package.]

Terraform exposes one numeric type rather than separate integer and
floating-point types. Conversion to a provider's API representation may impose
additional range or precision restrictions. Those restrictions belong at the
appropriate conversion boundary; the expression language does not infer an
API's machine type merely from the spelling of a number.

### 5.2 Null and Empty Values

Null represents absence within the value model. An empty string, a zero, and an
empty collection are present values and remain distinct from null. At an
argument boundary, null generally means that no value was supplied, subject to
the schema, defaults, and conversion rules of that context.

The implementation distinguishes `cty.NullVal(type)` from `cty.NilVal`.
The former is a Terraform-representable value; the latter is a Go-level sentinel
used where no cty value is available. Confusing the two can turn an internal
absence of a result into an unintended language-level null.

Null is also not the same as unknown. A known null is a specific statement
about the value. An unknown can still have null among its possible results.
Under the normal plan compatibility rules, a planned null cannot become a
non-null value at apply without contradicting the plan. [Source:
[`internal/plans/objchange/compatible.go`][object-compatible].]

For input variables, `nullable = false` excludes a null value at the variable's
outer boundary. It does not recursively exclude nulls from all nested
collections or object attributes. Defaults also participate in how a null input
is handled. [Source: [input variables][variable-doc].]

### 5.3 Unknown Values

An unknown represents information that Terraform does not yet have. It normally
has a known type and may carry additional constraints, but neither is
unconditional: `cty.DynamicVal` has no concrete type, and an unknown without a
non-null refinement may later resolve to null.

Consider a subnet whose configuration uses the ID of a network that has not yet
been created. Terraform can still describe the subnet's desired arguments,
leaving the ID unknown. It need not invent a placeholder string and later hope
that replacing it leaves the rest of the calculation unchanged. This is the
role of unknowns in the plan/apply model described in
[*Unknown Values: The Secret to Terraform Plan*][unknown-values].

A compound value can be partly known. An object can have known attribute names
and an unknown attribute value; a tuple can have known length but unknown
elements. Core and providers must distinguish an unknown collection from a
known collection containing unknown values. Those representations support
different conclusions about shape and identity.

#### 5.3.1 Unknowns and the Programming Model

Unknowns are handled by the evaluator rather than exposed as promise objects
that configuration must await or inspect.

> **P9 (A) - Honest Unknown.** Evaluation must not substitute an unsupported
> concrete value for missing information. It may produce a known result only
> when the available information determines that result; otherwise it must
> retain an appropriate unknown or report a justified error.

For example, a result can be known even when an input contains unknown
components: the length of a tuple with three elements is three regardless of
the elements' values. Type information can also justify an error before a
value is known. Accessing an attribute of an unknown string is invalid because
strings do not have attributes.

> **P10 (A) - Knownness Is Not Observable.** Configuration cannot branch on
> whether an ordinary value is currently known. The value's availability to
> the runtime is not a separate input to the configuration's meaning.

An `is_known` predicate would let a configuration choose one desired value
during planning and another during apply solely because more information had
arrived. Terraform instead evaluates the same expression with a more complete
context.

P10 does not prohibit the host from inspecting knownness. Core must do so to
render plans, determine which checks can run, diagnose unknown instance keys,
and defer work in runtimes that support deferral. These decisions are not
values on which the configuration can branch. [Source:
[*Unknown Values*][unknown-values].]

Providers can improve plan precision by predicting a result when their API
contract justifies doing so. An attribute is not inherently "known only after
apply" merely because one provider version reports it that way. Conversely, an
unknown does not mean "unchanged from the prior state." Core cannot assume
equality that the provider has not established.
The discussion in [Terraform issue #30937][unknown-discussion] explains both
the opportunity for better prediction and the limits imposed by remote systems.

#### 5.3.2 Increasing Knowledge

> **P11 (A) - Monotonic Knowledge.** For the same expression and semantic
> context, refining its inputs should produce a compatible refinement of its
> result. A result already established as known must not become a different
> known value merely because previously missing information became available.

The context qualification excludes changes to configuration, external inputs,
or the underlying operation. This is a rule about learning more about the same
computation, not a claim that arbitrary evaluations at different times return
the same result.

The precision need not increase on every evaluation. Learning one attribute
may leave another result wholly unknown. The requirement is consistency with
the earlier information. Terraform's plan/apply checks enforce the corresponding
contract at important boundaries (§10.4); they do not constitute a proof of
every expression implementation.

cty's [compatibility policy][cty-compatibility] makes a related, separately
scoped promise: later library versions can return more precise results for
previously unknown operations when those results remain within the earlier
range. That cross-version policy should not be confused with Terraform's
within-run plan contract.

#### 5.3.3 Known Shape and Instance Expansion

The ordinary CLI workflow requires enough information during planning to
enumerate resource and module instances. For `count`, this means a known count.
For resource or module `for_each`, it means known map keys or known members of
a set of strings. Map values may remain unknown.

This requirement is stronger than the requirement for many ordinary expression
results. A `for` expression or a `dynamic` block can involve unknown structure
that is carried forward for later evaluation. Such nested configuration is not
a set of separately addressed resource instances.

An instance change needs an address; an argument value can often remain partly
unknown. The boundaries therefore impose different planning requirements.
Core also contains gated deferral machinery that represents incomplete
expansion without guessing instance keys (§8.3). In ordinary CLI `for_each`,
an unresolved instance set remains an error.
[Sources: [unknown-value discussion][unknown-discussion];
[`internal/instances`][instances].]

#### 5.3.4 Unpredictable Functions and External Inputs

Functions such as `uuid`, `timestamp`, and `bcrypt` cannot generally promise
the same concrete result during plan and apply. Their function bodies compute
normally, but Terraform lists them as `impureFunctions` and wraps them with
cty's `function.Unpredictable` in `PureOnly` evaluation scopes. The wrapper
returns an unknown instead of asserting a result too early. [Source:
[`internal/lang/functions.go`][language-functions].]

The scope restricts when an impure function is evaluated; it does not make the
function itself pure. Reporting an unknown preserves the plan's honesty while
allowing apply to supply the eventual value.

Filesystem functions introduce a different dependency. `file`, for example,
reads an external input that must remain available and suitable for the run.
Function calls do not establish resource graph dependencies, so they are not a
mechanism for waiting for another resource to generate a file. Changes to such
inputs can invalidate assumptions made during planning and may be reported by
later consistency checks. A design using external inputs must account for
their lifetime across the plan/apply boundary. [Sources:
[the `file` function][file-function]; [*Unknown Values*][unknown-values].]

### 5.4 Refinements

A refinement records a constraint on an unknown: non-nullness, a string prefix,
a numeric bound, or bounds on collection length. It preserves useful knowledge
without asserting a concrete value.

> **P12 (D) - Refinement Soundness.** A refinement must describe a range that
> contains every result still legitimately possible. Additional knowledge may
> narrow that range, but must not exclude a valid eventual result.

Refinements belong to the value's possible meaning. They can enable deductions:
knowing that an unknown is non-null can resolve a null comparison, and knowing
collection shape can support further operations even when elements remain
unknown.

An operation may ignore an input refinement and compute a less precise but still
sound approximation. Some boundaries also discard refinements. This is distinct
from contradicting a refinement already committed to by a plan: losing internal
precision does not authorize an applied result outside the plan's permitted
range. Keeping those scopes separate reconciles optional refinement support with
P5 and P11.

Terraform produces refinements in language functions and checks unknown ranges
in `AssertValueCompatible`. The supported refinement kinds and their treatment
are documented in [cty's refinement guide][refinements]; the compatibility check
is in [`internal/plans/objchange/compatible.go`][object-compatible].

### 5.5 Marks: Sensitive and Ephemeral

Marks carry handling metadata with a value. Terraform uses them for sensitivity
and ephemerality. Unlike a refinement, a mark does not narrow the set of values
that an unknown might become. It describes how the value must be handled.

> **P13 (A) - Mark Propagation.** Evaluation must preserve the handling
> requirements of marked inputs in derived values unless an operation
> explicitly defines and justifies their removal.

Propagation is conservative because a general expression evaluator cannot
determine whether every derivation has removed all sensitive information.
Operations that temporarily unmark values for computation must preserve the
relevant marks or marked paths when reconstructing the result. Explicit
declassification, such as `nonsensitive`, is a separate language operation, not
an incidental consequence of conversion. [Sources:
[`internal/lang/marks`][marks]; [sensitive-data handling][sensitive-data].]

**Sensitive** controls disclosure in supported output surfaces. It does not
encrypt the value or exclude it from saved plans and state. Access to those
artifacts must therefore be controlled independently. A storage backend may
provide encryption, but that protection is not supplied by the sensitive mark.

**Ephemeral** excludes a value from Terraform's persisted plan and state value
storage. Ephemeral variables and resources can supply temporary credentials or
other run-scoped inputs to contexts permitted to consume them. Such values
must be supplied or obtained again when a later phase needs them; the saved
plan is not their storage mechanism.

> **P14 (P) - Ephemeral Non-Persistence.** Terraform must not persist
> ephemeral values in plan or state storage. A context that contributes
> persistent values must reject ephemeral input unless its contract explicitly
> excludes that input from the persisted representation.

This is not a claim that an ephemeral value can never leave the process.
Permitted provider operations can consume it. The contract concerns Terraform's
storage and the allowed flow of values through the language; it is not a
substitute for provider-side handling or disclosure controls.

**Write-only attributes** define a provider-facing non-persistence boundary.
They can accept ordinary or ephemeral inputs, but their values are represented
as null in persisted resource data. Removing an ephemeral mark while retaining
its underlying value would not satisfy that contract. Core's ephemeral helpers
validate and strip write-only values at the relevant boundaries.
[Sources: [ephemeral values and variables][variable-doc];
[write-only arguments][write-only-doc];
[`internal/lang/ephemeral`][ephemeral-implementation].]

Set handling needs particular care. The schema validator rejects write-only
attributes nested inside set blocks or nested set attributes. Its implementation
notes that marks within sets are promoted to the containing set, preventing the
required per-attribute handling. Set element correlation imposes further
constraints (§10.3).
[Source: [`configschema/internal_validate.go`][schema-validation].]

### 5.6 Values at the Plan Boundary

The value model lets a plan distinguish commitments that would otherwise be
conflated:

| Representation | Information conveyed |
|---|---|
| Known value | The result is fixed within the applicable plan contract. |
| Null | Absence is known, subject to the surrounding schema's interpretation. |
| Unknown | Some part of the result is not yet determined. |
| Refined unknown | The result is not concrete, but its possible range is constrained. |
| Sensitive mark | Supported output must handle the value as sensitive. |
| Ephemeral mark | The value is restricted to contexts that do not persist it. |

These dimensions can coexist. A sensitive value can be unknown; an ephemeral
value can be concrete. A design should identify which dimension it needs before
adding new propagation or serialization behavior.

## 6. Addresses

Addresses connect configuration, plans, state, diagnostics, and operator-facing
commands. `internal/addrs` represents them as structured types with string
forms. The type identifies what an address denotes; a string alone may be
ambiguous without its context.

### 6.1 Declarations and Instances

The address package distinguishes module call paths from expanded module
instances, and resource declarations from resource instances.

| Address type | Meaning |
|---|---|
| `Module` | A static path through module calls, such as `module.app.module.db`. |
| `ModuleInstance` | A path that includes instance keys, such as `module.app["blue"]`. |
| `Resource` | A resource mode, type, and name relative to a module. |
| `ConfigResource` | A resource in a static module path, independent of expansion. |
| `AbsResource` | A resource within a particular module instance, before selecting a resource instance. |
| `AbsResourceInstance` | A resource instance within a particular module instance. |

Thus `module.app["blue"].aws_instance.server` can identify all `server`
instances within one module instance, while
`module.app["blue"].aws_instance.server[0]` selects one member. A singleton
instance has no printed key, so its string form alone does not always reveal
the distinction between a resource and an instance.

> **P15 (A) - Static/Dynamic Separation.** Configuration declarations and
> their expanded instances are distinct objects. Address-taking interfaces
> must specify whether they identify declarations, collections, or concrete
> instances; a managed remote object's state binding belongs to an instance.

Some operations deliberately accept more than one address kind, such as a
whole resource or a single instance. Their semantics must define how the broader
address expands. That is different from treating the address kinds as
interchangeable. [Source: [`internal/addrs`][addresses].]

### 6.2 Instance Keys

`NoKey` identifies a singleton instance, `IntKey` is used for `count`, and
`StringKey` is used for resource and module `for_each`. That `for_each` accepts
a map or a set of strings; it is not general sequence iteration. The iteration
rules for `dynamic` blocks are a separate matter.

Core also has `WildcardKey`, printed as `[*]`, for partially expanded address
representations used with deferral. It does not invent a concrete instance key
and is not an ordinary managed-object binding. More generally, a partial address
describes the unresolved part of expansion without claiming to enumerate its
members. [Sources: `internal/addrs/instance_key.go`; [`internal/instances`][instances].]

### 6.3 Provider Addresses

A provider **source address** identifies a provider implementation, such as
`registry.terraform.io/hashicorp/aws`. A provider **configuration address**
identifies a configured use of that provider. The latter has a module context
and may have an alias.

Within a module, `LocalProviderConfig` uses the module's local provider name and
alias. `AbsProviderConfig` uses the resolved provider source and module path.
Its module path is static; parsing rejects module instance indexes in provider
configuration addresses. [Source: `internal/addrs/provider_config.go`.]

Modules containing their own provider configurations are incompatible with
module-call `count`, `for_each`, and `depends_on`. This restriction is enforced
by provider validation. The static provider address model is relevant context,
but does not by itself explain all three restrictions, particularly
`depends_on`. Reusable modules should instead receive provider configurations
through the supported caller-to-child association (§13.3).
[Sources: [`internal/configs/provider_validation.go`][provider-validation];
[providers within modules][module-providers].]

### 6.4 Address Capabilities

Several interfaces describe uses of addresses. `Referenceable` identifies
subjects supported by reference resolution. `Targetable` supplies the containment
relationships needed by targeting. `UniqueKey` provides comparable keys for
maps and sets.

Adding a referenceable address requires corresponding resolution in the
evaluation context. Adding a targetable address requires a clear account of
what selecting it includes. Neither interface should be inferred from the
existence of a printable address.

Internal graph vertices are a wider category. Expansion and close nodes, for
example, participate in execution without being objects a configuration can
reference. P2 governs configuration evaluation dependencies; it does not require
all runtime bookkeeping to appear in the configuration namespace.
[Sources: `internal/addrs/referenceable.go`, `targetable.go`, and `unique_key.go`.]

---

# Part III - Runtime and Persistence

## 7. Graph Construction and Execution

### 7.1 Operation-Specific Graphs

Core builds execution graphs through ordered transformation pipelines. A
transformer can introduce vertices, attach configuration or state, connect
dependencies, or remove unnecessary work. The pipeline differs by operation:
a planning graph must discover changes, while an apply graph must execute
changes already described by a plan.

The builders combine several sources of information:

| Information | Purpose |
|---|---|
| Configuration | Declarations, expressions, references, and lifecycle settings |
| Prior state | Existing objects, including objects no longer declared |
| Provider schemas and associations | Interpretation and provider selection |
| Planned changes, during apply | Concrete instance operations to execute |
| Run options | Scope and mode, including targeting and destroy planning |

`PlanGraphBuilder` and `ApplyGraphBuilder` specify the transformation order.
`ReferenceTransformer` connects reference-derived evaluation dependencies.
`AttachDependenciesTransformer` records resource dependency information for
state and lifecycle use. Destroy and create-before-destroy transformations
construct the operation-specific ordering required by lifecycle semantics. [Sources:
[`graph_builder_plan.go`][plan-graph], [`graph_builder_apply.go`][apply-graph],
and [`transform_reference.go`][reference-transform].]

Transformation order matters because later stages rely on information attached
by earlier ones. A design that adds graph behavior must identify where that
information becomes available and how later pruning, targeting, and validation
treat it. Cycle detection is a property of the resulting graph, not of whether
an edge happened to originate in `ReferenceTransformer`.

Objects removed from configuration illustrate why configuration alone is
insufficient. Their state still supplies vertices and information needed to
plan their removal. Such objects are often called **orphans**: they lack a
current declaration, not a managed identity.

### 7.2 Dependency Direction

Core's graph edges represent dependencies. The walker uses those dependencies
to determine when a vertex is eligible to run. A diagram of internal edges
therefore need not point in the order operations execute.

For simple resource creation, if `B` depends on `A`, `A` must be ready before
`B` can use it. For destruction of both, the required operation order is
normally `B` before `A`. Core constructs the appropriate destroy relationships;
it does not achieve this merely by traversing the unchanged create graph
backwards. [Source: [resource destruction notes][destroying].]

`TransitiveReductionTransformer` removes edges whose ordering is already
implied by another path. Reachability is preserved, but the final direct edge
set need not contain one edge for every original reference. Code examining
the graph must distinguish direct edges, reachability, and source references.

### 7.3 Reconstructing the Apply Graph

The in-memory plan records changes and the information needed to interpret
them; it is not a serialized adjacency graph. `ApplyGraphBuilder` reconstructs
execution ordering from the plan's changes together with configuration and
state. `DiffTransformer` supplies operation vertices for recorded instance
changes.

A saved plan archive also contains configuration and state information.
Consequently, saying that the change records do not contain graph edges does
**not** mean the archive contains no ordering information. It contains inputs
from which ordering is reconstructed. [Sources: [`internal/plans/plan.go`][plan-model];
[`internal/plans/planfile`][plan-files]; [apply graph builder][apply-graph].]

This distinction matters for any behavior that must survive saving a plan and
applying it in another process. Runtime-only information from the planning walk
cannot be assumed to remain available. It must either be represented in the
saved artifact or be reproducible from the artifact's inputs.

### 7.4 Walking and Dynamic Subgraphs

The graph walker permits independent vertices to make progress concurrently.
Core limits execution parallelism; the ordinary CLI default is ten, adjustable
with `-parallelism`. A dependency establishes a readiness constraint, not a
promise about the relative start times of unrelated operations.

Diagnostics are associated with work performed during the walk. A failed
dependency normally prevents dependent work from running, while independent work
and required cleanup can still proceed. The `AlwaysRunVertex` mechanism supports
vertices that must run despite upstream failures. This is one reason an apply
can make partial progress before reporting an error.

Some vertices implement `GraphNodeDynamicExpandable`. During the walk they
produce subgraphs after the information needed for expansion becomes available.
Those subgraphs are validated and walked as part of the operation. Static
reference analysis precedes the dependent evaluation, while concrete instance
graphs are added as the walk progresses.
[Sources: [`internal/dag/walk.go`][graph-walker];
[`internal/terraform/graph.go`][terraform-graph].]

## 8. Instance Expansion

### 8.1 Registration and Enumeration

`instances.Expander` coordinates module and resource repetition. After the
repetition expression for a particular context has been evaluated, Core
registers a singleton, count-based, or `for_each` expansion. Other parts of the
runtime can then enumerate the corresponding instances.

The implementation provides registration methods such as `SetModuleCount` and
`SetResourceForEach`, and enumeration methods such as `ExpandModule` and
`ExpandResource`. Expansion modes are represented separately in
`expansion_mode.go`. The ordering requirement is explicit: a caller must register
the relevant expansion before asking the expander for its instances.
[Source: [`internal/instances`][instances].]

These are not two global passes that necessarily finish for the whole
configuration at once. Expansion is coordinated with the graph walk: a child
module's resource repetition may depend on values available only after that
module instance has been established.

### 8.2 Nested Expansion

Every instance of a module uses the same static declarations, but can receive
different input values. A resource inside the module can therefore expand
differently in each module instance. The total number of resource instances is
the total across those individual expansions, not necessarily one uniform
resource count multiplied by the number of modules.

The static address model allows Core to analyze references before all those
instance keys are known. Instance-aware logic is then used where the operation
requires a concrete subject. Static analysis and instance execution use
different levels of address granularity according to the work being performed
(§6.1, §7).

Changes to repetition also affect existing objects. If a key disappears from a
known desired instance set, the object bound to that key remains in state until
its planned removal is carried out. Expansion therefore participates in normal
deletion and replacement behavior as well as creation.

### 8.3 Unknown Expansion and Deferral

In the ordinary CLI plan/apply workflow described here, an unknown `count` or
an unknown resource or module `for_each` instance set is an error. Core cannot
produce the required concrete instance changes. An unknown attribute within a
known instance does not present the same addressing problem.

Core also contains gated deferral support, used by Stacks. Unknown expansion
modes, partial addresses, and deferred-change records allow the runtime to
retain information about work that it cannot yet plan completely. This
machinery is not ordinary CLI behavior in the version-specific account of this
paper. [Sources: [`internal/instances`][instances];
[`internal/plans`][plans-package]; [Stacks runtime][stacks].]

Deferral separates **executable planned changes** from **work requiring another
planning round**. The latter can include a resource whose instance set cannot
yet be enumerated. Its representation must identify the unresolved scope
without pretending to supply concrete instance addresses.

The current plan does not authorize execution of that deferred work. A later
round must determine and plan it. Thus deferral changes the completeness of a
single round, not value fidelity (P5) or plan authorization (P24). It also leaves
the configuration's meaning independent of knownness (P10): the runtime reports
incomplete work rather than making the configuration choose a different desired
result.

Consumers of partial plans need an explicit completion model. They must be able
to distinguish "all desired work is complete" from "the executable portion of
this round succeeded." Reporting, approval, persistence, and orchestration all
need to preserve that distinction. The design background is discussed in
[Terraform issue #30937][unknown-discussion].

## 9. The Resource Instance Lifecycle

### 9.1 Current, Deposed, and Tainted Objects

A resource instance can have one **current** object and multiple **deposed**
objects. A deposed object is an earlier object still tracked after another
object has taken its place as current. Create-before-destroy replacement
requires this representation because the old and new remote objects can coexist
at one logical resource instance address.

Deposed keys distinguish those retained objects within the instance. They are
not additional user-selected `count` or `for_each` keys.
[Source: [`internal/states/resource.go`][state-resource].]

**Tainted** is a status indicating that Terraform cannot treat an object as
ready and complete, commonly after a partial creation failure. A tainted
object is normally planned for replacement rather than accepted as a stable
realization of configuration. Taint can also be set through explicit state
operations.
For an operator-requested replacement, `-replace` expresses the intent as a
plan option instead of first editing the object's status.
[Sources: [`internal/states/instance_object.go`][state-object];
[planning behaviors][planning].]

### 9.2 Replacement

Replacement combines removal of the old object with creation of a new one.
The ordinary lifecycle has two orderings:

| Ordering | Consequence |
|---|---|
| Delete then create | The old object is removed before the replacement is created. |
| Create then delete | The replacement can become current while the old object remains tracked as deposed. |

`create_before_destroy` selects the second ordering when replacement is needed.
It does not itself require replacement. Replacement can instead follow from
provider requirements, taint, `replace_triggered_by`, or an operator's
`-replace` request. Core combines the reason for replacement with the applicable
lifecycle ordering. [Sources: [planning behaviors][planning];
[lifecycle reference][lifecycle-doc].]

Create-before-destroy also requires the remote system to permit temporary
coexistence. Naming rules, capacity, and uniqueness constraints remain the
provider and configuration author's concern. An ordering preference cannot make
two objects coexist when the API forbids it.

### 9.3 Lifecycle Dependencies

Create-before-destroy can affect dependencies, not just the resource on which it
is written. If a dependent must remain alive until its replacement is ready,
destroying one of its dependencies too early would contradict that lifecycle.
Core propagates the required create-before-destroy behavior and rewrites the
operation graph accordingly.

`ForcedCBDTransformer` propagates the setting where required.
`CBDEdgeTransformer` constructs the corresponding edge changes. Saved lifecycle
metadata preserves relevant behavior when an object later has no configuration.
[Sources: [`transform_destroy_cbd.go`][cbd-transform];
[resource destruction notes][destroying].]

The resulting operation can interleave creation, dependent updates, and removal
of deposed objects. A single-resource description of replacement is therefore
only the starting point. The destruction notes give concrete dependency
diagrams for these combinations; they should be consulted before changing
lifecycle ordering.

## 10. The Core/Provider Contract

### 10.1 Domain Independence

Core understands resource modes, addresses, values, schemas, and operation
contracts. A provider understands how those operations map to an external
system: which API calls are required, which changes need replacement, and which
representations are semantically equivalent.

> **P16 (A) - Core Is Domain-Agnostic.** Core's resource behavior is expressed
> through domain-independent contracts. Knowledge specific to an infrastructure
> API belongs in the provider and must reach Core through an explicit protocol.

Domain independence does not mean that Core knows nothing about resource
behavior. Its protocol deliberately exposes concepts such as replacement,
identity, and deferred work. The separation concerns who interprets the
external system and how that interpretation is communicated.
[Sources: [architecture overview][architecture]; [provider protocol][protocol].]

### 10.2 Managed-Resource Operations

The provider interface divides a resource's lifecycle into operations with
different inputs and obligations:

| Operation | Role |
|---|---|
| `UpgradeResourceState` | Convert stored data from an older schema to the current schema without treating the conversion as refresh. |
| `ReadResource` | Observe the remote object and report its current state. |
| `ValidateResourceConfig` | Diagnose configuration; do not rewrite it. |
| `PlanResourceChange` | Predict the result of a proposed change within the configuration and prior-state rules. |
| `ApplyResourceChange` | Perform the change and return the resulting known state. |
| `ImportResourceState` | Obtain initial state information for an object to bind to an address. |
| `MoveResourceState` | Perform a supported provider-side state conversion for a move between resource types. |

This is a division of responsibilities, not a universal fixed RPC sequence.
For a changing instance, Core can call `PlanResourceChange` during planning and
again during apply after upstream values are known. It checks the later result
against the earlier plan. Replacement, no-op, deletion, and other paths differ;
"exactly twice per run" is not a reliable call-count contract.

Validation must tolerate information that is legitimately unknown at the point
of the call. A provider can reject an error already established by type or known
values, but cannot generally reject an optional value merely because it is not
yet known. Later evaluation provides an opportunity to validate the resolved
configuration.

Successful applied state, refreshed state, and upgraded state must not introduce
unknown values into persisted state. This requirement concerns representation,
not complete observation of every remote fact (§11.1).
[Sources: [resource-instance change lifecycle][resource-lifecycle];
[`internal/providers/provider.go`][provider-interface].]

### 10.3 Proposed New State and Collection Correlation

Before provider planning, Core computes a proposed new object from configuration
and prior state. `ProposedNew` and its helpers encode how schema flags affect
that merge.

At the attribute level, a non-computed value normally comes from configuration.
For an attribute marked computed and left null in configuration, the prior value
is the usual starting point. The provider can then choose an appropriate
planned value within the protocol rules. The starting proposal must not be
confused with a final promise that the prior value will remain unchanged.

Nested optional-and-computed attributes have an additional case. If the prior
nested value contains non-computed content, Core can infer that it previously
depended on configuration. `optionalValueNotComputable` handles this case so
that removing the configuration need not preserve the whole prior object.
That rule is specific to the nested schema; it is not a generic rule for every
`Optional+Computed` scalar. [Source:
[`internal/plans/objchange/objchange.go`][object-change].]

The Plugin Framework adds its own ordered planning steps around the provider's
implementation: defaults, unknown marking for computed values, attribute plan
modifiers, and resource plan modifiers. These operate within Core's consistency
contract. A modifier such as `UseStateForUnknown` is appropriate only when the
provider can justify retaining the old value; it is not a general way to hide
uncertainty. [Source: [Framework plan modification][framework-planning],
supplied v1.18 documentation.]

Nesting mode determines how old and new elements are related. Lists provide
positions, maps provide keys, and single objects can be compared recursively.
Sets provide value-based identity rather than a separate stable key. A changed
set element can look like one value disappearing and another appearing.

Core therefore correlates set elements heuristically. `validPriorFromConfig`
asks whether a prior element could have arisen from a configuration element
with computed content filled in. This is useful but less informative than
explicit keys. Provider schema design must take account of which values define
element identity and which are expected to change. This limitation concerns
nested collection correlation, not the instance-key model of resource
`for_each`.

### 10.4 Plan Validity and Applied Compatibility

Core checks different relationships at different boundaries.

**`AssertPlanValid`** checks the provider's planned object against configuration
and prior state. Configured non-null attributes must normally retain their
configured values. A provider may retain the corresponding prior value when the
new spelling is functionally equivalent; Core can check the value relationship,
but relies on the provider to judge semantic equivalence. Computed values have
different latitude when configuration leaves them unset. Write-only values
must be null in the planned representation.

Nested-block validation depends on nesting mode and schema flags. Configured
list and map blocks have structural correspondence rules. Set blocks allow
less complete correlation. The cited implementation also has separate handling
for computed blocks and unknown `dynamic` block results. These details are
version-sensitive and must be read from the applicable schema and protocol.
Nested blocks are not independent resource graph nodes, so their restrictions
cannot be explained by simply equating block count with resource instance count.
[Source: [`internal/plans/objchange/plan_valid.go`][plan-valid].]

**`AssertObjectCompatible`** checks whether a later object is compatible with
the planned object. Known values must remain equal. Unknowns can become known
within their type and refinement constraints. Collection structure and
correlation affect how the comparison is made; sets require different treatment
from indexed or keyed collections. [Source:
[`internal/plans/objchange/compatible.go`][object-compatible].]

The corresponding diagnostics include "Provider produced invalid plan" and
"Provider produced inconsistent result after apply." They report a broken
contract, not merely an unusual result. Their location also limits what they
can establish: a mismatch can originate in earlier evaluation or changing
external inputs, and an error after an RPC does not undo remote work already
performed.

### 10.5 Provider-Defined Functions

Provider-defined functions extend expression evaluation, not the resource
lifecycle. Their contract requires deterministic results for the same arguments
and no observable side effects. Core can detect some inconsistent results,
but cannot prove purity from the behavior it observes.

This distinction separates two provider extension points. A resource operation
is expected to interact with an external system according to its lifecycle
contract; a function is expected to compute a value. Putting an operation behind
function syntax would not give it resource planning, state, or recovery
semantics. [Sources: [provider-defined functions][provider-functions-doc];
[provider protocol][protocol].]

Functions use the namespace `provider::name::function`. Keeping provider-defined
names separate from built-ins reduces collisions as either side evolves.
The compatibility implications of shared namespaces are discussed in §16.3.

### 10.6 Legacy Compatibility

Provider protocol versions 5 and 6 coexist, with clients in `internal/plugin`
and `internal/plugin6`. Protocol version alone does not describe every
compatibility behavior. In particular, `legacy_type_system` supports restricted
concessions for providers built with the original SDK.

Those concessions are not permission for new implementations to disregard the
provider contract. They preserve interoperability with legacy value handling
and are accompanied by normalization and validation paths, including
`NormalizeObjectFromLegacySDK`. New schema features must account for whether
the participating protocol and SDK can represent them correctly.
[Sources: [provider protocol definitions][protocol];
[`internal/plans/objchange/normalize_obj.go`][legacy-normalization].]

For design work, the useful distinction is between the intended modern contract
and the compatibility path that preserves older behavior. Both must be
understood, but neither should be silently substituted for the other.

## 11. State

### 11.1 Bindings and Known Values

State records the association between a Terraform resource instance and the
object it manages. Attribute values support planning and refresh, but cannot
replace that association. A remote API may enumerate objects without knowing
which Terraform configuration address is responsible for each one.

> **P17 (A) - One Address, One Object.** The management model assumes one
> resource instance address for each managed remote object and at most one
> current object at an instance address. Replacement may temporarily retain
> additional, deposed objects at that instance.

The current-object slot is part of Core's state representation. Uniqueness of
remote ownership is a broader modeling obligation: Core cannot generally
recognize that two addresses, even within one state, identify the same remote
object. It also has no global registry across independent states. A configuration
must not rely on duplicate management being detected automatically.
[Sources: [state purpose][state-purpose]; [`internal/states`][states].]

> **P23 (A) - State Is Wholly Known.** Persisted state snapshots contain no
> unknown values. In-memory planning representations can carry unknown planned
> values, but those must not be mistaken for persisted observations.

Concrete state does not mean complete knowledge of an object. Providers record
only what their schemas and API access support; write-only values are excluded.
Null may represent absence according to the schema, not a promise that Core
has discovered every fact about the remote system.

The implementation makes the distinction explicit. `ObjectPlanned` represents
transient placeholders during planning. State encoding cannot preserve their
unknowns, so expression evaluation must consult the corresponding planned value
when it needs that information. This internal encoding behavior is not license
for a provider to return unknown applied state.
[Sources: [`internal/states/instance_object.go`][state-object];
[discussion in #30937][unknown-discussion].]

State also records lifecycle and interpretation metadata: dependencies,
create-before-destroy status, schema versions, provider-private data, and
resource identity. Dependencies use configuration-resource addresses so that
relevant relationships remain available after a declaration is removed.
[Source: `internal/states/instance_object_src.go`.]

### 11.2 Snapshots, Lineage, and Interfaces

A state file includes the state snapshot and metadata about its history.
`Serial` identifies successive modifications within that history.
`Lineage` identifies the history itself. Serial numbers are meaningful for
comparison only when the lineages match; a greater serial from an unrelated
lineage is not a newer version of the same state.

Lineage is opaque. Consumers should compare it for equality, not infer meaning
from its spelling. State readers also distinguish format versions and diagnose
unsupported formats rather than interpreting them as a known version.
[Sources: [`internal/states/statefile`][state-files].]

The raw state file and the documented JSON output of `terraform show -json`
are different interfaces. The existence of external tools that read raw state
does not make every internal field a supported public API. Automation should
use the documented interface appropriate to its task and respect its format
version. State backends and migration tools have additional responsibilities
that are not defined merely by the JSON output schema.
[Sources: [JSON output format][json-format];
[compatibility promises][compatibility].]

### 11.3 Refresh and Drift

Refresh asks providers for updated observations of managed objects. The
provider must distinguish a material change from an equivalent normalization.
If a remote API returns equivalent JSON with different whitespace, retaining
the prior spelling can avoid a meaningless difference. If the remote value has
actually changed, preserving the old value would hide drift.

Core cannot make that semantic distinction for an arbitrary domain. The
`ReadResource` contract therefore gives the provider responsibility for it.
[Source: [resource-instance change lifecycle][resource-lifecycle].]

The plan retains two relevant snapshots. `PrevRunState` represents the previous
run's recorded state; `PriorState` represents the refreshed state used for
planning. Comparing them supports drift reporting, while comparing desired
configuration with the refreshed state supports the proposed changes. These
are different questions, even when displayed in the same plan.
[Source: [`internal/plans/plan.go`][plan-model].]

Updated observations must also reach downstream evaluation. A resource can
require no corrective action while one of its observed attributes has changed.
Outputs and other expressions referring to that attribute must see the
refreshed value rather than a stale copy.

Refresh-only mode lets the operator review the proposed state and output updates
without planning ordinary changes to remote objects. The standalone
`terraform refresh` command is deprecated in favor of the reviewable
`terraform apply -refresh-only` workflow. This is a change in how state updates
are approved, not the point at which Terraform first began reading remote state
during planning. [Source: [refresh command documentation][refresh-doc].]

### 11.4 Coordination and Partial Failure

State coordinates successive operations on managed bindings, but is not a
transaction log capable of rolling back every provider's external effects.
When apply partly succeeds, its resulting state must preserve the progress
Terraform can report. Discarding that progress would make subsequent planning
operate from an account known to be obsolete.

Provider responses and state persistence can themselves fail. Recovery must
distinguish a failed remote operation, an incomplete account of its result, and
a failure to save state that Core already holds. These are different failure
boundaries; a single "apply failed" message is not enough to infer the state of
the external system.

Backend locking, where supported and enabled, coordinates cooperating operations
using the same state. Saved-plan checks also reject plans whose state history
no longer matches the assumptions under which they were produced. Neither
mechanism locks the remote world or prevents another independent state from
managing the same object. Disabling locking or performing manual state
operations changes the coordination assumptions and requires corresponding
care. [Sources: [state locking][state-locking]; [`internal/backend`][backends].]

### 11.5 Resource Identity

Structured resource identity gives a provider an object-typed representation
for identifying a remote object, separate from its ordinary attributes.
The state representation stores identity data and an identity schema version;
the provider protocol includes identity schemas and upgrade operations.
Configuration-driven import can use an `identity` object.

This is useful when an API's identity is naturally composite, such as an
account, region, and name. It avoids making a single import string the only
representation of that structure. Structured identity does not remove the
distinction between a Terraform address and a remote identity: the state
binding still associates the two. Nor does it automatically establish global
uniqueness of management.
[Sources: [import blocks][import-doc]; [provider protocol][protocol];
`internal/states/instance_object_src.go`.]

## 12. Destruction and Dependency Lifetimes

### 12.1 The Reversal Principle

For a simple dependency, destruction reverses the normal creation requirement:
if `B` needs `A`, remove `B` before removing `A`. This applies during routine
configuration changes as well as a full `terraform destroy`.

> **P18 (D) - Destroy Is Reversal.** Resource removal derives its dependency
> constraints from the relationships used to manage the objects. Simple
> destruction reverses their creation order; mixed operations and lifecycle
> settings require the corresponding operation-specific graph transformations.

The short name is a mnemonic, not an algorithm for reversing every edge in the
runtime graph. Provider configuration, expansion, cleanup, replacement, and
dependent updates have different lifetime requirements. Their vertices cannot
all be treated as if they were ordinary resource creations.
[Source: [resource destruction notes][destroying].]

### 12.2 Configured and Recorded Dependencies

`DestroyEdgeTransformer` connects destruction with other planned operations
using both current configuration relationships and recorded dependencies.
Recorded information is especially important for removed declarations and
deposed objects, whose original configuration may no longer be available.

Provider lifetime is part of the same problem. A provider configuration may
depend on an object that is itself being destroyed. Core must retain the
information and ordering necessary to configure and use the provider until its
remaining operations are finished. Create-before-destroy introduces additional
constraints on when old objects can be released.

These interactions explain why removal behavior must be designed alongside
creation and update behavior. A declaration's disappearance is a normal input
to Terraform, not an exceptional condition under which its dependencies and
provider association can be forgotten. [Sources:
[`transform_destroy_edge.go`][destroy-transform];
[`transform_destroy_cbd.go`][cbd-transform]; §14.3.]

---

# Part IV - Composition and Evolution

## 13. Modules

### 13.1 Scope Without an Independent Transaction

A module is a unit of authorship, distribution, configuration scope, and reuse.
Its inputs and outputs expose selected values while keeping internal names
private. It does not, by default, create an independent plan/apply transaction.

> **P19 (P) - Modules Are Namespaces.** Module boundaries organize
> configuration and control visibility. A child module participates in its
> caller's overall Core operation rather than executing as an isolated
> lifecycle unit.

Core can evaluate independent resources in different modules concurrently.
A module output can become available once its own dependencies are satisfied,
without waiting for every unrelated resource in that module. Consequently,
module-level diagrams can appear circular even when the actual value and
resource dependencies are acyclic: two modules can exchange values if those
particular calculations do not depend cyclically on one another.

This is sometimes described as flattening the module tree. The description
concerns evaluation semantics, not the literal absence of module-related graph
vertices. The implementation has module expansion and close nodes, and uses
dynamic subgraphs. A module boundary also does not imply separate persisted
state for that child. [Sources: [architecture overview][architecture];
[`internal/terraform`][terraform-runtime].]

### 13.2 Inputs, Outputs, and Dependencies

Ordinary values cross module boundaries through input variables and output
values. A caller cannot directly traverse into a child module's resource
namespace, and a child cannot directly name arbitrary objects in its caller.
Expressions supplying inputs and consuming outputs establish the corresponding
references.

An explicit `depends_on` on a module call expresses a broader dependency than
passing a particular value. It can delay work throughout the child module,
including reads that could otherwise occur earlier. Where a dependency is
already expressed by the relevant input and output references, those references
give Core more precise information than a whole-module dependency.
[Source: [`depends_on` reference][depends-on-doc].]

Type constraints, nullability, and sensitive or ephemeral declarations define
additional parts of the interface. They determine what values a module accepts,
what information survives conversion, and what handling obligations cross the
boundary. Modules therefore compose through more than matching attribute names.
Their contracts include the value semantics described in §5.

### 13.3 Provider Association

Provider configurations cross module boundaries through a dedicated association
mechanism, not as ordinary input-variable values. A reusable child module
declares its provider requirements; the caller supplies suitable configurations.
Default configurations can be inherited, while aliases require explicit
association through the `providers` meta-argument.

Local provider names are interpreted within each module. The association must
resolve to a compatible provider source even when callers and children use
different local names. Version requirements and configured instances also have
different roles: selecting an implementation does not itself supply credentials,
region settings, or an alias configuration.

Root-owned provider configurations are the normal design for reusable modules.
Legacy child modules with their own provider blocks remain supported with
restrictions, including the incompatibility with module-call `count`,
`for_each`, and `depends_on` described in §6.3. Provider configurations must also
remain available while objects associated with them still need operations,
including removal. [Source: [providers within modules][module-providers].]

### 13.4 Custom Conditions

Variable validation, preconditions, postconditions, and `check` blocks express
author-supplied assertions. Their placement determines what they can observe
and what a failure prevents.

Resource preconditions run after instance expansion and before evaluation of
the resource's configuration arguments. They can therefore use `count.index`
or `each` information, but cannot guard the repetition expression that was
needed to establish the instance. Postconditions check resulting information
and can prevent dependent work from proceeding. Unknown condition inputs can
delay a check until enough information is available.
[Sources: [custom conditions][conditions-doc]; [lifecycle reference][lifecycle-doc].]

Failure severity is part of the contract. A failed precondition or postcondition
blocks the applicable operation. An ordinary `check` assertion failure is a
warning and does not serve as an execution gate. Terraform Test incorporates
these results into its own assertion and expected-failure handling; its outcome
must not be inferred directly from the ordinary CLI warning severity.
[Sources: [check blocks][check-doc]; [`internal/moduletest/run.go`][test-run].]

These constructs let module authors document and enforce assumptions close to
the values and objects involved. Separately enforced governance policy has a
different authority boundary: an assertion that an author can remove from the
same configuration is not an independent approval control.

## 14. Refactoring and Management Transitions

### 14.1 Addresses Change While Objects Remain

Without refactoring information, renaming a resource can look like removal of
one declaration and addition of another. State still associates the existing
object with the old address. A `moved` block tells Core how to reinterpret that
binding before ordinary planning.

An ordinary address move can preserve the remote object. It does not guarantee
that the remainder of the plan will be a no-op: the new configuration may also
require an update or replacement. Supported moves between resource types can
ask a provider to convert state. It is therefore inaccurate to describe every
move, or every run containing one, as making no provider calls.
[Sources: [moved blocks][moved-doc]; [`internal/refactoring`][refactoring].]

The related constructs have distinct purposes:

| Construct | Management transition |
|---|---|
| `moved` | Associate an existing binding with a new configuration address. |
| `removed` | End management of the selected object or objects, with the declared removal behavior. |
| `import` | Establish management of an existing remote object at an address. |

For `removed`, the default behavior includes destruction. Setting
`lifecycle { destroy = false }` instead removes the state binding without
destroying the remote object. Import can call the provider and read the remote
object; it is not an API-free assignment of an address to arbitrary data.
[Sources: [removed blocks][removed-doc]; [import blocks][import-doc].]

### 14.2 Persistent Declarations, Not Repeated Commands

> **P20 (P) - Refactoring Statements Are Inert.** A refactoring declaration
> records a relationship between configuration and management history. Once
> that transition has been satisfied, retaining the declaration must not by
> itself repeat the completed operation.

"Inert" describes this relationship to history. It does not mean that
processing the declaration can never cause provider operations or state
changes. Invalid or contradictory declarations can also produce diagnostics
rather than silently doing nothing.

This property allows a module to retain a sequence of `moved` declarations so
that users upgrading from different prior versions can reach the current
address structure. The declaration becomes part of the module's migration
contract. Removing it too early can break an upgrade path even though the
current configuration is otherwise unchanged.

Core validates move relationships for conflicts and cycles and orders valid
move chains before applying them to the working state. The relevant code is in
`move_validate.go` and `move_execute.go`. Configuration-driven transitions can
then be shared in version control and reviewed through a plan.
Imperative state commands remain separate administrative operations; they do
not acquire those properties merely because they can achieve a similar final
binding. [Sources: [`internal/refactoring`][refactoring];
[module refactoring documentation][module-refactoring].]

### 14.3 Metadata Lifetime

A removed declaration may have contained information needed to remove its
object: provider association, dependency relationships, and optional
destroy-time configuration. The desired-state declaration can disappear before
the remote object does.

> **P21 (P) - Metadata Lifetime.** Information required by an object's
> remaining operations must survive until those operations are complete,
> including when the declaration that originally supplied it has been removed.

State preserves some of that information, including dependencies and lifecycle
metadata. Provider configurations may need to remain in configuration until
their objects are removed. A `removed` block can retain removal-specific intent
and configuration without continuing to declare the object as desired.
The lifetime argument is developed in the *removed blocks* design proposal;
current behavior is described in the [removed-block reference][removed-doc].

This does not imply that all former configuration should be copied into state.
Some information is unsuitable for persistence, particularly ephemeral values.
The design must distinguish what is stored, what can be reconstructed, and what
must be supplied again. The same lifetime analysis applies to deposed objects,
provider cleanup, and recovery after a partially completed operation.

## 15. Related Languages and Runtimes

HCL syntax and cty values are shared by several Terraform-related systems.
Shared libraries do not imply identical evaluation, state, or approval
boundaries. This section identifies the relationships needed to reason about
cross-cutting features; it is not a complete specification of each language.

### 15.1 Terraform Test

Terraform Test adds `run` blocks that plan or apply a configuration and evaluate
assertions. Mocks and overrides let tests replace selected provider or module
behavior. The test framework invokes the normal Core runtime for the operation
under test, while supplying its own orchestration, evaluation contexts, state
management, and result handling.

Calling Test "the same runtime" is therefore only partly informative. The
resource behavior is exercised through Core, but setup, assertions, expected
failures, and cleanup also belong to the testing layer. A cross-cutting feature
must specify which parts run normally and how tests can supply inputs and
observe outcomes. [Sources: [Terraform tests][tests-doc];
[`internal/moduletest`][module-test].]

### 15.2 Stacks

Stacks is an orchestration layer over trees of Terraform modules. Its address,
configuration, plan, state, and runtime packages have roles analogous to the
corresponding Core packages, while component evaluation invokes the module
runtime beneath them.

A component is a distinct planning and state boundary in a way that an ordinary
child module is not. Stacks coordinates those component operations and supports
deferred work when information is insufficient for a complete round. The
executable changes still have a plan contract; orchestration determines how
later planning rounds complete the remaining work.
[Source: [`internal/stacks/README.md` and runtime packages][stacks].]

The distinction is architectural, not an exemption from compatibility.
Different boundaries allow different contracts, but those contracts still have
consumers. A rule about a single module-tree operation cannot simply be applied
to an entire Stack without identifying the corresponding scope.

### 15.3 Query

Query uses provider-backed `list` operations to discover existing objects and
can support generation of configuration for their subsequent management.
The query configuration has its own decoding and command integration, while
reusing provider and planning infrastructure.

Discovery is distinct from the state binding that establishes ownership.
A data source can also return a collection or an empty result, so the
distinction is not "data reads one existing object, Query reads many." It lies
in the discovery and configuration-generation contract of `list`, compared with
the provider-defined value returned by a data source.
[Sources: `internal/configs/query_file.go`;
[`internal/command/query.go`][query-command].]

### 15.4 Actions

An `action` block describes a provider-defined operation. Declaring an action and
invoking it are separate parts of the language. Invocation can be requested
directly or associated with a resource lifecycle trigger.

Actions have their own invocation protocol rather than the persistent
object-management contract of a managed resource. An action declaration should
therefore be understood through its documented invocation behavior, not assumed
to have resource attributes, refresh, or convergence semantics merely because
it shares provider configuration and repetition mechanisms.
[Sources: [action blocks][action-doc]; [invoking actions][invoke-actions-doc];
`internal/configs/action.go`.]

### 15.5 Policy Integration

Governance policy evaluates acceptability under rules whose authority can be
separate from the module author. Its relationship to Core includes the values
and metadata made available for evaluation, the point at which evaluation
occurs, and how enforcement results affect the operation.

At the cited source revision, `internal/policy` defines a client with setup and
resource, provider, and module evaluation requests. Requests distinguish current
attributes, prior attributes, metadata, and redaction information.
[Source: [`internal/policy/policy.go`][policy-client].]

This section covers the Core-side integration. Release availability, language
semantics, and deployment requirements must be established from the policy
product's own versioned documentation.

### 15.6 Shared Features, Explicit Boundaries

A new value mark, provider capability, or artifact field may have consumers in
several of these systems. Each consumer needs an intentional treatment:
inherit the Core behavior, translate it at a defined boundary, or reject the
unsupported use with an appropriate diagnostic.

Identical syntax is not always desirable, and different syntax is not
automatically a semantic divergence. The useful comparison is the contract:
what is evaluated, which state it uses, what is planned, what is approved, and
what is persisted. Appendix B includes these questions in the interaction
checklist.

## 16. Compatibility

### 16.1 The Documented Promise

Terraform's v1.x compatibility promises cover a substantial part of the
language, a specified set of CLI workflows, provider communication, and
installation protocols. They aim to preserve valid configurations and protected
automation across v1.x upgrades. Provider-specific behavior is independently
versioned and is not covered by Core's promise.

For automation, the documented interfaces include JSON output modes and exit
status codes. Natural-language output and logs are not stable parsing
interfaces. Compatibility also does not imply that an older Terraform release
can read every newer state format or understand a newly introduced language
feature. [Source: [v1.x compatibility promises][compatibility].]

The published policy includes important qualifications:

| Area | Qualification |
|---|---|
| Validity | The promise generally concerns configurations that can plan and apply without errors. |
| Invalid configuration | Error handling may change; errors can be detected earlier. |
| Implementation bugs | Documented behavior may take precedence, with care to limit compatibility impact. |
| Experiments | Experimental features may change or be removed. |
| Existing deprecations | Explicit deprecation cycles may end during v1.x. |
| Exceptional changes | Critical security issues or changes in external dependencies can justify changes outside the usual expectation. |

Diagnostic changes must be assessed by their effect on protected behavior.
The policy permits changes to invalid-configuration handling and prefers earlier
detection; phase alone does not determine compatibility.

### 16.2 Supported Behavior Becomes a Contract

> **P22 (P) - Supported Behavior Is a Contract.** Within the documented
> compatibility scope, valid configurations and supported interfaces create
> obligations for later versions. A design must account for the behaviors it
> permits, not only the examples it intends to encourage.

This principle does not make every implementation detail permanent.
It asks designers to identify which choices become observable commitments.
Accepting several forms of an argument, exposing an address syntax, or allowing
a combination of features can create dependencies that are difficult to
discover later.

Restricting an initially unsupported combination can preserve room for a later,
well-defined extension. Once a combination is supported, narrowing it requires
attention to compatibility rather than merely a simpler implementation.
Explicit errors and documented boundaries are useful when the intended
semantics are not yet settled. This is one of the principal design lessons of
the supplied *Hitchhiker's Guide to Language Design*.

### 16.3 Namespaces

Some Terraform namespaces combine Core-defined names with names supplied by
modules or providers. A new resource meta-argument can collide with a provider's
existing argument. A new module meta-argument can collide with an input
variable. Similar concerns arise where predefined expression roots share syntax
with resource type names.

These collisions must be considered before reserving a new word. A name that is
unused in Core is not necessarily unused in the ecosystem. The discussion of
`enabled` in [#21953][conditional-resources] gives a concrete account of the
module and provider compatibility concern without requiring any conclusion
about the feature's other merits.

Separate namespaces can reduce that exposure. Provider-defined functions use
`provider::name::function`, so adding a built-in function does not reserve a
provider's function name. The broader namespace analysis also appears in the
*Language Editions* proposal.

### 16.4 Editions and Versioned Semantics

The *Language Editions* proposal explores selecting language semantics per
module so that different editions could coexist in a configuration. That is
design background, not a general upgrade mechanism available to practitioners.
At the cited revision, the parser recognizes only the default `TF2021` edition.
[Source: [`internal/configs/experiments.go`][editions-parser].]

A future edition mechanism would need to define what changes at a module
boundary and what remains shared, including values, provider association,
addresses, and artifacts. A new syntax selector alone would not resolve those
cross-version interactions. Stacks defines a different orchestration boundary
and must be evaluated against its own compatibility commitments.

### 16.5 Evolution of the Model

Terraform has added capabilities while retaining the plan/apply model, typed
partial values, and persistent management bindings. Those foundations explain
how features can fit together, but do not make the current set of abstractions
the only possible design.

A useful architectural reference makes assumptions explicit enough to examine.
When a design preserves them, the existing contracts help explain its behavior.
When it changes them, the work includes identifying affected consumers and
establishing the replacement contract. That is the role of the numbered
principles and dependency table, rather than a presumption that an earlier
decision must remain unchanged.

---

# Appendices

## Appendix A - Principle Index and Architectural Dependencies

The table summarizes the numbered principles; their full statements and
qualifications are in the referenced sections. **A**, **D**, and **P** mean
architectural axiom, derived guarantee, and design policy, as defined in §0.2.
The labels do not rank the seriousness of a contract violation.

**Dependencies in this model** identifies supporting assumptions in the account
given here. It is not a necessary-and-sufficient dependency relation, an
exhaustive list of implementation dependencies, or a proof that replacing an
assumption makes the supported property impossible. A blank entry means that
the table does not identify another numbered principle as support.

| ID | Kind | Name | Summary | Section | Dependencies in this model |
|---|---|---|---|---|---|
| P1 | A | Data Flow | Configuration describes relationships; Core determines evaluation order. | 1.2 | - |
| P2 | A | Reference Order | References alone establish configuration evaluation dependencies. | 1.2 | - |
| P3 | A | Pure Evaluation | Expression computation is distinct from managed-resource operations. | 2.1 | - |
| P4 | P | Closed Action Vocabulary | Core defines the operation classes of its protocols. | 2.2 | P16 |
| P5 | D | Value Fidelity | Applied values must satisfy the plan's known values and unknown ranges. | 2.3 | P3, P9, P11, P12 |
| P6 | D | Convergence | A complete successful apply should stabilize the desired result under stable conditions. | 3.1 | P5, P17 |
| P7 | A | Schema-Directed Interpretation | Schemas complete the interpretation of syntactically distinct arguments and blocks. | 4.1 | - |
| P8 | P | One Evaluation Semantics | Restricted contexts share expression semantics while limiting inputs and capabilities. | 4.3 | P3, P11 |
| P9 | A | Honest Unknown | Missing information is represented without unsupported guesses. | 5.3.1 | - |
| P10 | A | Knownness Is Not Observable | Configuration cannot choose a result by inspecting runtime knownness. | 5.3.1 | - |
| P11 | A | Monotonic Knowledge | Refining inputs preserves previously established information in the same semantic context. | 5.3.2 | P3, P9, P10 |
| P12 | D | Refinement Soundness | A refinement must include every still-valid eventual result. | 5.4 | P9 |
| P13 | A | Mark Propagation | Derived values retain handling requirements unless removal is explicitly defined. | 5.5 | - |
| P14 | P | Ephemeral Non-Persistence | Ephemeral values must not enter persisted plan or state value storage. | 5.5 | P13 |
| P15 | A | Static/Dynamic Separation | Declarations, instance collections, and concrete instances have distinct address roles. | 6.1 | - |
| P16 | A | Core Is Domain-Agnostic | Providers communicate API-specific knowledge through domain-independent contracts. | 10.1 | P7 |
| P17 | A | One Address, One Object | The management model requires unique ownership, with one current object per instance. | 11.1 | P15 |
| P18 | D | Destroy Is Reversal | Simple destruction reverses resource dependencies; mixed operations require lifecycle transformations. | 12.1 | P2, P21 |
| P19 | P | Modules Are Namespaces | Modules provide scope and composition without independent lifecycle transactions. | 13.1 | P1, P2, P15 |
| P20 | P | Refactoring Statements Are Inert | Keeping a satisfied historical declaration does not repeat the completed transition. | 14.2 | P15, P17 |
| P21 | P | Metadata Lifetime | Required metadata survives until the operations that need it are complete. | 14.3 | P17 |
| P22 | P | Supported Behavior Is a Contract | Permitted behavior creates obligations within the documented compatibility scope. | 16.2 | - |
| P23 | A | State Is Wholly Known | Persisted snapshots contain no unknowns; planning representations are different. | 11.1 | - |
| P24 | D | Plan Authorization | Resolving unknowns does not authorize changes beyond the current plan. | 2.3 | P4, P15 |

For example, P18 cites P21 because destruction needs dependency information
after configuration has changed. P14 cites P13 because propagating the ephemeral
mark is part of enforcing its storage boundary. P6 additionally needs stable
inputs, adequate scope, and conforming provider behavior; those conditions are
not exhausted by the numbered entries in its row.

The table is useful for tracing a design question into other sections. It does
not replace the explanation of why a particular proposal affects a particular
contract.

## Appendix B - Feature Interaction Checklist

The checklist covers recurring architectural questions. Its purpose is to
organize investigation, not to certify completeness by checking every row.

| Area | What the design should establish |
|---|---|
| Evaluation dependencies | Identify referenced subjects and their resolution rules. Distinguish those dependencies from internal lifecycle, provider-lifetime, and cleanup ordering. |
| Values | Specify type conversion, null handling, partial values, refinements, and mark propagation. Keep runtime decisions about unknowns separate from configuration-observable behavior. |
| Planning | Identify the authorized changes and value commitments. Define whether planning must be complete and, if not, how deferred work is reported and subsequently planned. |
| Saved artifacts | Explain how apply reconstructs required information in a new process. Identify what is saved, recomputed, or supplied again, including ephemeral inputs. |
| Expansion and addresses | Distinguish declarations from instances. Define address forms, repetition, empty expansions, unknown expansions, and containment used by targeting or refactoring. |
| Lifecycle | Cover creation, update, replacement, deposed objects, and removal. Include removal caused by changed instance keys or deleted declarations, not just full destroy. |
| State and identity | Specify which bindings and metadata change, what partial failure leaves behind, and how schema upgrades and state coordination apply. Do not assume newer storage must be readable by all older versions. |
| Providers | Keep API-specific knowledge behind an explicit contract. Define capability negotiation and behavior with unsupported protocol or SDK versions, including useful diagnostics. |
| Composition | Describe module inputs, outputs, provider association, and assertion behavior. Identify any broader dependency introduced by the feature. |
| Related runtimes | Determine how Test, Stacks, Query, and other applicable consumers exercise or reject the capability. Shared parsing does not establish shared lifecycle semantics. |
| User and automation interfaces | Cover native and JSON configuration, human-readable presentation, documented JSON outputs, exit behavior, and the environments intended to support the feature. |
| Compatibility | Identify newly reserved names, newly supported combinations, and changes to valid configurations or protected workflows. Assess diagnostic changes by their actual effect rather than by text alone. |

Evidence should include the relevant implementation paths and representative
interactions, especially across phases. Testing an isolated successful create
path cannot establish a saved-plan contract, removal behavior, or preservation
of sensitive and ephemeral handling.

## Appendix C - Scope Notes

Several boundaries recur throughout the paper. They are collected here to
prevent the short principle names from being read more broadly than their
definitions.

| Boundary | Scope |
|---|---|
| Expression computation and graph operations | P3 does not claim that planning has no external interactions. Refresh, data-source reads, and ephemeral-resource operations are distinct from expression computation. |
| Partial operations | Targeting and deferral limit what a round establishes about the whole configuration. They do not themselves authorize unplanned changes or contradict the executable plan's known values. |
| External inputs | Filesystem contents and other changing inputs need a lifetime model. P11 concerns refined knowledge of the same computation, not arbitrary reevaluation against a changed world. |
| Provisioners | Provisioners run configured commands under their documented lifecycle rules. Their command effects are not modeled as ordinary provider resource attributes, and resource value checks are not a specification of those effects. |
| Legacy providers | Compatibility paths preserve particular older behaviors. They are not the default contract for new provider implementations. |
| Related languages | The principles apply at identified boundaries. A Stack component operation and a complete Stack are not the same scope; neither is a test run and its surrounding test orchestration. |

Provisioner behavior is documented in the [provisioner reference][provisioners-doc].
The other boundaries are developed in the sections cited by their principle
numbers.

## Appendix D - Limits of This Reference

This paper does not provide a complete operational specification for backends,
provider SDKs, or the related language family. Three areas require more detailed
references for implementation work.

**Persistence and recovery:** backend-specific locking and write guarantees,
saved-plan validation, cancellation, and recovery when remote progress cannot
be saved successfully.

**Provider conformance:** exact protocol rules for each operation, retry and
timeout behavior, normalization, identity upgrades, and the relationship between
SDK conveniences and Core's checks.

**Runtime applicability:** a systematic mapping of individual features to Test,
Stacks, Query, policy integrations, and their supported deployment environments.
The architectural comparisons in §15 do not supply that matrix.

These are limits of this paper's coverage, not claims that the corresponding
documentation or implementation does not exist. Where a design depends on one
of these details, the applicable versioned source remains necessary.

## Appendix E - Sources and Further Reading

### Core Design Documents

All implementation links use the revision identified in §0.4.

| Source | Main use in this paper |
|---|---|
| [`docs/architecture.md`][architecture] | Subsystem roles and the relationship between configuration, graphs, providers, and state |
| [`docs/planning-behaviors.md`][planning] | Default resource planning and configuration-, provider-, and run-driven adjustments |
| [`docs/resource-instance-change-lifecycle.md`][resource-lifecycle] | Values and obligations across resource RPCs |
| [`docs/destroying.md`][destroying] | Dependency ordering for removal and replacement |
| [`docs/plugin-protocol`][protocol] | Wire definitions and provider operation contracts |

The design documents explain intended behavior. The corresponding implementation
supplies operation-specific paths and details of versioned capabilities.

### Implementation Map

`internal/configs` and [`configschema`][schemas] cover configuration decoding
and schemas. [`internal/addrs`][addresses] defines address kinds.
`internal/lang`, [`marks`][marks], and [`ephemeral`][ephemeral-implementation]
cover evaluation and handling metadata. [`internal/plans/objchange`][object-change-package]
contains proposed-state construction and consistency checks.

[`internal/terraform`][terraform-runtime], [`internal/dag`][dag-package], and
[`internal/instances`][instances] contain graph construction, walking, and
expansion. [`internal/states`][states] and [`statefile`][state-files] contain
state representations and serialization. [`internal/refactoring`][refactoring]
handles historical address and management transitions. Provider interfaces and
clients are in `internal/providers`, `internal/plugin`, and `internal/plugin6`.

[`internal/stacks`][stacks], [`internal/moduletest`][module-test], query
configuration and command code, and [`internal/policy`][policy-client] show
how related systems use or extend those mechanisms. Their package boundaries
are implementation evidence, not a substitute for each system's public contract.

### Versioned Documentation and Value Libraries

The supplied Terraform **v1.16.x (RC)** documentation covers expressions,
blocks, modules, lifecycle, state, tests, and CLI workflows. Its
[v1.x compatibility promises][compatibility] define the scope discussed in §16.
The supplied Plugin Framework **v1.18.x** documentation provides the provider
author's account of [plan modification][framework-planning] and related resource
behavior.

[HCL][hcl] supplies structural and expression syntax.
[cty][cty] supplies the value model, with separate documents on
[refinements][refinements], [marks][cty-marks], and
[compatibility][cty-compatibility]. These libraries have their own versions and
contracts. Their general capabilities must be distinguished from the subset
and integration provided by a particular Terraform release.

### Essays and Technical Discussions

Martin Atkins' essays provide the principal public explanation of the language
model used here:

- [*Evolving the Terraform Language*][evolving-language], 1 March 2019:
  the v0.12 redesign and its design preferences.
- [*Terraform is a Data Flow Language*][data-flow], 19 August 2019:
  data-flow evaluation and its relationship to declarative programming.
- [*Unknown Values: The Secret to Terraform Plan*][unknown-values],
  14 June 2021: partial evaluation and the plan's value commitments.

[Terraform issue #21953][conditional-resources], including
[James Bardin's explanation of static references][static-references], discusses
conditional instances, reference resolution, and namespace compatibility.
[Issue #30937][unknown-discussion] discusses unknown values, provider
prediction, state knownness, and possible deferral. Issue comments provide
technical rationale and historical context; they are not, by themselves,
release specifications.

### Internal Design Background

The supplied *Hitchhiker's Guide to Language Design*, dated 15 July 2026,
motivates the emphasis on compatibility and feature interactions.
The [`hashicorp/terraform-proposals`][proposal-collection] collection contains
related design work, including *removed blocks* on metadata lifetime and
*Language Editions* on namespace overlap and versioned semantics.
[*Everything is a Plan*][everything-plan] provides additional background on
reviewable changes.

These internal materials explain design arguments. A proposal's presence in the
repository does not establish its adoption, release status, or authorship by any
particular individual. Current behavior is grounded in the documentation and
implementation cited alongside the relevant claim.

[action-doc]: https://developer.hashicorp.com/terraform/language/block/action
[addresses]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/addrs
[apply-graph]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/terraform/graph_builder_apply.go
[architecture]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/docs/architecture.md
[backends]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/backend
[cbd-transform]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/terraform/transform_destroy_cbd.go
[check-doc]: https://developer.hashicorp.com/terraform/language/block/check
[compatibility]: https://developer.hashicorp.com/terraform/language/v1-compatibility-promises
[conditional-resources]: https://github.com/hashicorp/terraform/issues/21953
[conditions-doc]: https://developer.hashicorp.com/terraform/language/validate
[core]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb
[cty]: https://github.com/zclconf/go-cty
[cty-compatibility]: https://github.com/zclconf/go-cty/blob/main/COMPATIBILITY.md
[cty-marks]: https://github.com/zclconf/go-cty/blob/main/docs/marks.md
[dag-package]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/dag
[data-flow]: https://log.martinatkins.me/2019/08/19/terraform-data-flow-language/
[depends-on-doc]: https://developer.hashicorp.com/terraform/language/meta-arguments/depends_on
[destroy-transform]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/terraform/transform_destroy_edge.go
[destroying]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/docs/destroying.md
[editions-parser]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/configs/experiments.go
[ephemeral-implementation]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/lang/ephemeral
[everything-plan]: https://github.com/hashicorp/terraform-proposals/blob/16e9d3f07fe8c798d7848b89cf8985bee5f6197d/everything-is-a-plan/everything-is-a-plan.md
[evolving-language]: https://log.martinatkins.me/2019/03/01/terraform-language/
[file-function]: https://developer.hashicorp.com/terraform/language/functions/file
[framework-planning]: https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification
[graph-walker]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/dag/walk.go
[hcl]: https://github.com/hashicorp/hcl
[import-doc]: https://developer.hashicorp.com/terraform/language/block/import
[instances]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/instances
[invoke-actions-doc]: https://developer.hashicorp.com/terraform/language/invoke-actions
[json-format]: https://developer.hashicorp.com/terraform/internals/json-format
[json-syntax]: https://developer.hashicorp.com/terraform/language/syntax/json
[language-functions]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/lang/functions.go
[legacy-normalization]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/objchange/normalize_obj.go
[lifecycle-doc]: https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle
[marks]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/lang/marks
[module-doc]: https://developer.hashicorp.com/terraform/language/block/module
[module-providers]: https://developer.hashicorp.com/terraform/language/modules/develop/providers
[module-refactoring]: https://developer.hashicorp.com/terraform/language/modules/develop/refactoring
[module-test]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/moduletest
[moved-doc]: https://developer.hashicorp.com/terraform/language/block/moved
[object-change]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/objchange/objchange.go
[object-change-package]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/objchange
[object-compatible]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/objchange/compatible.go
[plan-actions]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/action.go
[plan-files]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/planfile
[plan-graph]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/terraform/graph_builder_plan.go
[plan-model]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/plan.go
[plan-valid]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans/objchange/plan_valid.go
[planning]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/docs/planning-behaviors.md
[plans-package]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/plans
[policy-client]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/policy/policy.go
[proposal-collection]: https://github.com/hashicorp/terraform-proposals
[protocol]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/docs/plugin-protocol
[provider-functions-doc]: https://developer.hashicorp.com/terraform/language/functions#provider-defined-functions
[provider-interface]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/providers/provider.go
[provider-validation]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/configs/provider_validation.go
[provisioners-doc]: https://developer.hashicorp.com/terraform/language/provisioners
[query-command]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/command/query.go
[refactoring]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/refactoring
[reference-transform]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/terraform/transform_reference.go
[refinements]: https://github.com/zclconf/go-cty/blob/main/docs/refinements.md
[refresh-doc]: https://developer.hashicorp.com/terraform/cli/commands/refresh
[removed-doc]: https://developer.hashicorp.com/terraform/language/block/removed
[resource-lifecycle]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/docs/resource-instance-change-lifecycle.md
[schema-validation]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/configs/configschema/internal_validate.go
[schemas]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/configs/configschema
[sensitive-data]: https://developer.hashicorp.com/terraform/language/manage-sensitive-data
[stacks]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/stacks
[state-files]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/states/statefile
[state-locking]: https://developer.hashicorp.com/terraform/language/state/locking
[state-object]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/states/instance_object.go
[state-purpose]: https://developer.hashicorp.com/terraform/language/state/purpose
[state-resource]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/states/resource.go
[states]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/states
[static-references]: https://github.com/hashicorp/terraform/issues/21953#issuecomment-1422894604
[terraform-graph]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/terraform/graph.go
[terraform-runtime]: https://github.com/hashicorp/terraform/tree/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/terraform
[test-run]: https://github.com/hashicorp/terraform/blob/05cecbb315e87abc2c2bdb182963dd8e36d574fb/internal/moduletest/run.go
[tests-doc]: https://developer.hashicorp.com/terraform/language/tests
[type-constraints]: https://developer.hashicorp.com/terraform/language/expressions/type-constraints
[unknown-discussion]: https://github.com/hashicorp/terraform/issues/30937
[unknown-values]: https://log.martinatkins.me/2021/06/14/terraform-plan-unknown-values/
[variable-doc]: https://developer.hashicorp.com/terraform/language/block/variable
[write-only-doc]: https://developer.hashicorp.com/terraform/language/manage-sensitive-data/write-only
