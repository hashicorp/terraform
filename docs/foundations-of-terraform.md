# The Foundations of Terraform

### Language semantics, runtime contracts, and architectural dependencies

**Status:** Draft  
**Audience:** Terraform Core engineers, architects, and language designers  
**Classification:** Internal

---

## Abstract

Terraform combines a data-flow configuration language with a runtime that plans
and applies changes to external systems. Its behavior rests on several
representations working together: references establish evaluation dependencies,
typed values carry partial knowledge, addresses distinguish declarations from
instances, and state records the objects Terraform manages. Providers connect
these domain-independent mechanisms to particular remote systems.

This paper describes those foundations in four layers — the programming model,
the configuration and value representations, the execution machinery, and the
mechanisms for composition and evolution. Three distinctions recur throughout:
expression evaluation versus provider operations, a plan's value commitments
versus its authorized changes, and configuration structure versus the instances
created from it. Numbered principles and an architectural dependency table give
design discussions something concrete to cite. They are an account of the
system's assumptions and contracts, not a formal proof of the implementation.

The purpose is to support reasoning across feature boundaries, because a
proposed extension has to be understood in terms of the assumptions it uses and
the assumptions it changes — including the consequences for planning, state,
composition, and compatibility.

## 0. Scope and Conventions

### 0.1 Architectural Reasoning

A Terraform feature rarely belongs to one subsystem. An expression determines
instance keys; those keys become addresses in a plan and in state; references to
those addresses establish evaluation dependencies; and saved dependency
information orders destruction after the configuration changes. Local behavior
has to be understood in the context of the whole system.

This paper's method rests on one claim: **an objection that identifies a
conflict with an established architectural premise deserves an answer, even when
the reviewer cannot name the specific future failure.** The reviewer's job is to
identify the premise and show where the proposal conflicts with it. The
designer's job is to respond in one of three ways — demonstrate that the premise
still holds, replace it along with the contracts that depend on it, or define a
separate contract with an explicit boundary. Whichever route the design takes, a
changed guarantee has to be visible to the people relying on it.

This is a way of examining designs, not a claim that Terraform's architecture is
fixed, and a conflict does not prove that failure is inevitable. It identifies
work the design must account for. The discussion of interacting language
features in the supplied *Hitchhiker's Guide to Language Design* is the direct
motivation.

### 0.2 Three Kinds of Claim

The numbered principles use three labels:

| Label | Meaning in this paper |
| --- | --- |
| **Axiom (A)** | An architectural assumption on which the present model is built. |
| **Derived guarantee (D)** | A property supported by those assumptions and by the implementation contracts described alongside it. |
| **Design policy (P)** | A deliberate choice about the language or its supported behavior. |

The labels are explanatory, not a formal classification of the source code. An
axiom here is not a mathematical axiom, and a derived guarantee is not a theorem
proved from the other numbered statements. Guarantees come with conditions:
convergence, for example, requires stable inputs and a provider capable of
implementing the desired result.

Nor do the labels rank importance. Ephemeral non-persistence is a policy because
it is a chosen language contract, and violating it is no less serious for that.
Any proposed change has to consider what was promised to users and to other
components, whatever label this paper attaches.

Appendix A records architectural dependencies: assumptions that help explain or
support a principle. The entries are neither exhaustive nor
necessary-and-sufficient, and removing one assumption does not prove that every
possible implementation of the dependent property is impossible.

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
Framework **v1.18.x** trees. These are distinct baselines, so a capability
present in the implementation is not necessarily available in the documented CLI
release; version-sensitive behavior is flagged where it matters.

The source types are not interchangeable. Source links identify files at that
revision, with symbol names to locate the relevant implementation inside them.
Public documentation links are reading aids, while the supplied versioned trees
are the documentation baseline. Essays and issue discussions explain rationale.
Proposals describe possible designs, and adoption has to be established
separately.

Two terms are used throughout in a specific sense. **Core** means Terraform's
evaluation and planning runtime and its supporting packages, as distinct from
the CLI command layer, providers, and surrounding automation. **Expression
evaluation** means computing a value from an expression and its evaluation
context — narrower than a **graph walk**, which can also call providers and
update working state.

The principles describe the Terraform configuration language and Core. Related
languages may use Core while adding evaluation or orchestration rules of their
own (§15), and their contracts have to be considered separately.

---

# Part I - Programming Model

Three ideas underpin everything that follows. References rather than statement
order determine what happens when. A plan makes two separable promises, one
about values and one about scope. And convergence — the property most users
have in mind when they describe what Terraform does — holds only under
conditions worth naming precisely.

## 1. Data Flow and References

### 1.1 Declarative Programming

Terraform is a programming language in the declarative tradition. A
configuration describes the objects an author wants and how they relate, not a
procedure for building them. That much is true of many languages, and
"declarative" by itself settles very few design questions.

The sharper description is **data flow**. Expressions say how values are
obtained and combined; references say which values depend on which. Core reads
those references to decide what may be evaluated when. A single argument
carries both meanings at once:

```hcl
resource "aws_subnet" "example" {
  vpc_id = aws_vpc.main.id
}
```

The subnet's `vpc_id` takes its value from the VPC's `id`, and because it does,
Core must deal with the VPC first. Martin Atkins develops this distinction in
[*Evolving the Terraform Language*][evolving-language] and
[*Terraform is a Data Flow Language*][data-flow].

### 1.2 Evaluation Order

The constructs that look most like control flow all turn out to compute values
instead of sequencing steps. A conditional expression selects between two
values. A `for` expression builds one collection from another. A `dynamic`
block builds nested configuration, and `for_each` on a resource or module
selects which instances exist. Each of these changes the work Terraform has to
do; none of them tells Terraform in what order to do it.

> **P1 (A) - Data Flow.** A Terraform configuration describes objects, values,
> and their relationships. Core derives an evaluation order from those
> relationships rather than executing the configuration as an ordered program.

> **P2 (A) - Reference Order.** Evaluation order is determined solely by
> references. A dependency requiring one configuration object to be evaluated
> before another must be expressed through a reference.

Reading P2 correctly depends on separating a reference from a value. A
reference names a subject. In `aws_instance.example.id` the reference is
`aws_instance.example`; the trailing `.id` is a traversal that selects
information out of whatever value that subject has. Only the first part creates
a dependency, which is why the second part is optional:

```hcl
depends_on = [aws_instance.example]
```

Here the reference contributes no argument value at all. It contributes the
dependency and nothing else. `depends_on` is an application of P2 rather than
an exception to it.

Because the graph is built from references, Core has to find them before it can
evaluate anything, and it finds them by reading syntax rather than results.
A conditional therefore cannot hide a reference in the branch it does not take:
`var.enabled ? aws_instance.a.id : "none"` depends on `aws_instance.a` whatever
`var.enabled` turns out to be. James Bardin explains how this static analysis
feeds graph construction in [Terraform issue #21953][static-references].

P2 governs evaluation order. The execution graph carries more than evaluation:
it also sequences provider startup and shutdown, instance expansion, and the
relative order of creates and destroys. Lifecycle transformations and
dependencies recorded in state add constraints that no expression mentions
(§7, §12). Both kinds of ordering are real; only the first comes from
references.

### 1.3 A Useful Analogy

A spreadsheet shows the difference between declaring a dependency and choosing
an execution order. A formula names the cells it uses, and the calculation
engine works out the schedule and detects cycles. Moving a formula further down
the sheet does not make it run later.

Terraform adds concerns a spreadsheet does not have: persistent object
identity, provider operations, partial knowledge, and an approval step before
managed-resource changes. The analogy explains the evaluation model and stops
there. The other data-flow systems in Atkins' essay are useful within the same
limit. [Source: [*Terraform is a Data Flow Language*][data-flow].]

### 1.4 Declarations and Conditional Instances

A declaration and the instances it produces are different things. Setting
`count = 0` does not remove the declaration; it gives the declaration an empty
collection of instances, and the name still resolves:

```hcl
resource "aws_instance" "example" {
  count = 0
}

output "ids" {
  value = aws_instance.example[*].id # [] - valid, and empty
}
```

Static analysis still sees `aws_instance.example`, so the reference is legal.
Evaluation finds no instances, so the result is an empty list.

That is why repetition is more than syntactic convenience: it gives absence a
value-level representation while keeping the configuration namespace stable.
A proposal for genuinely conditional declarations would have to answer two
questions this design already answers — how references to an absent declaration
resolve, and what value such a declaration has — and it cannot borrow the
answers from an empty instance collection. The discussion in
[#21953][conditional-resources] works through the distinction.

Checking whether a remote object exists is a separate concern. Some data
sources require their object to exist and report absence as an error; others
return collections, empty results, or computed information. Those behaviors
belong to the provider's data-source contract. Reading an object and deciding
which configuration owns its lifecycle remain different operations (§11).

### 1.5 Language-Design Preferences

The v0.12 design work stated three preferences: favor readers over writers,
prefer explicit behavior, and keep simple tasks simple while allowing complex
ones. They explain a good deal about expression syntax and the regularity of
the language, and they do not decide every tradeoff on their own.

Familiar shorthand, for instance, can stay useful even after a more explicit
form exists. The question to ask is whether the shorthand has a coherent
meaning and composes with the rest of the language — not whether each construct
maximizes one preference in isolation. [Source:
[*Evolving the Terraform Language*][evolving-language].]

## 2. Planning and Applying

### 2.1 Expressions and Operations

Terraform keeps two things apart: computing a value and changing a remote
object. Expressions produce the desired values. Core compares those values with
prior and refreshed state, and providers help turn the comparison into proposed
changes. Apply then attempts those changes within the plan's constraints.

> **P3 (A) - Pure Evaluation.** Expression computation is distinct from
> resource operations. Evaluating an expression is not an imperative request
> to perform a managed-resource change; those changes belong to the runtime's
> planning and application protocol.

The scope of that claim matters, because planning is far from inert. A planning
run can refresh managed resources, read data sources, and open ephemeral
resources; initialization and state operations have effects of their own; and a
few functions read external inputs or produce unpredictable results and need
special handling (§5.3.4). P3 draws a line between expression computation and
those operations. It does not confine every external interaction to apply.

Apply evaluates configuration too. It does not replay a self-contained script
of final values captured at plan time. As unknown inputs become known, Core
re-evaluates the configuration that depends on them and checks that the results
still agree with the plan.

The guidance in [`docs/planning-behaviors.md`][planning] prefers the plan/apply
process for externally visible changes, while acknowledging the operations that
already sit outside it.

### 2.2 Core's Action Model

Everyday managed-resource changes fit a small vocabulary: create, update,
delete, replace, no-op, and read for data sources. Replacement carries an
ordering choice with it, normally delete-then-create or create-then-delete.

The full enumeration in `plans.Action` is wider. Alongside the familiar cases
it includes `Forget`, which drops a state binding without touching the remote
object, the paired `ForgetThenCreate` and `CreateThenForget`, and `Open`,
`Renew`, and `Close` for ephemeral resources. CRUD is a good introduction to
the model and a poor inventory of it. [Sources: [planning behaviors][planning];
[`internal/plans/action.go`][plan-actions].]

> **P4 (P) - Closed Action Vocabulary.** Core defines the action classes used
> by its planning and execution protocols. Providers implement and influence
> those operations; they do not independently add action classes to Core's
> resource-change model.

A provider has real influence inside that vocabulary. It can declare that a
changed attribute forces replacement, or return a more precise planned value
than Core could compute on its own. What it cannot do is invent a new kind of
change: Core decides how the resulting action is represented, ordered, and
approved. A provider capability earns lifecycle semantics by fitting an
explicit Core protocol, not by being reachable through a plugin.

The planning design document distinguishes three common sources of special
behavior:

| Source | Scope | Examples |
| --- | --- | --- |
| Configuration | Follows the module or resource declaration | `ignore_changes`, `create_before_destroy`, `moved` |
| Provider | Reflects the remote system's requirements | Replacement requirements and planning adjustments |
| Run options | Applies to the operator's particular invocation | `-replace`, `-refresh-only`, `-target` |

The categories overlap in practice, and their value is in naming who owns a
decision and how long it lasts. A configuration setting travels with the module
and appears in review. A run option belongs to a single invocation, which means
every UI wrapping Core has to expose it before operators can use it.
[Source: [planning behaviors][planning].]

### 2.3 What a Plan Commits To

A plan makes two kinds of promise at once, and separating them heads off a
recurring confusion. It says what the values will be, and it says which changes
are allowed to happen.

> **P5 (D) - Value Fidelity.** Within the planned change contract, a value
> recorded as known must remain equal when the change is applied. An unknown
> value may become known, but the result must satisfy the constraints recorded
> for that unknown.

> **P24 (D) - Plan Authorization.** Applying a plan is limited to the changes
> that plan authorizes. Resolving unknown values does not authorize additional
> subjects or a different class of change.

P5 is a consistency rule, enforced by the compatibility checks in
`internal/plans/objchange`. If the plan shows `instance_type = "t3.micro"`,
apply must produce `t3.micro`. If it shows `(known after apply)`, apply may
produce any value the recorded constraints permit.

P24 is about scope rather than content. It is why an unknown value cannot
quietly enlarge the blast radius: learning a VPC's real ID lets Core finish the
subnet it already planned, and gives it no license to replace a database the
operator never saw in the plan.

Both rules are checked at Terraform's own boundaries, which bounds what they
can establish. Core can compare the values a provider returns against the
values it promised. It cannot verify that the provider made the right API calls
to get there. A plan describes Terraform-level change, not a trace of provider
implementation.

Neither rule promises success. An API can reject a change, credentials can
expire, a provider can return an inconsistent result. Core's obligation is then
to report the failure rather than quietly substitute a different proposal — and
a reported failure says nothing about whether remote changes already happened
(§3.2, §11.4).

Deferral introduces a third property alongside these two: **completeness**.
A partial plan can leave work for a later round without loosening either its
value commitments or the scope of what it authorizes. Deferred work is simply
not authorized by the current plan, and a later round must plan it (§8.3).

## 3. Convergence and Its Limits

### 3.1 Stable Results

Terraform aims to converge managed objects on the declared configuration.
Under stable conditions, applying the proposed changes should leave nothing
further to propose for that same desired result.

> **P6 (D) - Convergence.** Given stable configuration, inputs, and relevant
> external conditions, a complete successful apply should leave the managed
> objects consistent with the desired result, so that a subsequent plan
> proposes no further changes to them.

Every qualifier in that sentence is doing work. An input that changes on its
own, such as `timestamp()`, produces a genuinely new desired value on the next
run, and the resulting plan is correct rather than defective. A provider may
also have to absorb API normalization or eventual consistency before a remote
object can be represented stably at all.

The recognizable symptom of failure is a plan that proposes the same change
after every apply. A perpetual diff is good evidence that configuration,
provider planning, applied results, and refreshed state disagree somewhere, and
poor evidence about which of them is wrong. The contracts that narrow it down
are in the [resource-instance lifecycle][resource-lifecycle] and the provider
planning rules (§10).

### 3.2 Scope, Drift, and Partial Failure

Terraform looks at external systems during an operation and at no other time.
Whatever happens between runs — a console edit, an autoscaler, another pipeline
— surfaces as **drift** at the next refresh, however convergent the previous
apply was. Refresh is a fresh observation, not a retroactive verdict on the
earlier run.

Scope limits convergence a second way. A configuration owns some of the objects
a provider can see, not all of them, and `-target`, deferral, or an interrupted
apply can leave part of even that subset untouched. Such a run establishes
nothing about whole-configuration convergence, even if it happens to end with
no further changes to propose.

Apply is also not a transaction across remote systems. Early changes commit
before a later one fails, and there is no general rollback. Terraform records
what it can and reports the errors; recovery proceeds forward from that updated
state (§11.4).

### 3.3 Representational Requirements

The contracts in Part I set requirements the rest of the system has to meet,
and each one explains a representation described next. Dependency analysis must
work before values are known, so references have to be resolvable independently
of what they evaluate to. A plan must record what it can and cannot promise, so
the value model needs a first-class representation of missing information.
A change must name a concrete object, so addresses have to distinguish a
declaration from the instances it produces. And management must survive across
runs and configuration revisions, so state has to carry identity and lifecycle
information that the current configuration may no longer mention.

---

# Part II - Configuration, Values, and Addresses

Part I's contracts impose requirements, and three representations meet them:
the configuration language authors write, the value model that carries partial
knowledge through evaluation, and the addresses that let every subsystem name
the same object. Most cross-cutting design questions turn out to be questions
about one of these three.

## 4. The Configuration Language

### 4.1 Syntax and Schema

HCL gives Terraform two languages in one file. The structural language
describes bodies, blocks, and arguments; the expression language computes the
values those arguments take. Terraform v0.12 unified them, replacing the older
split between HCL structure and HIL string interpolation.
[Source: [*Evolving the Terraform Language*][evolving-language].]

In native syntax, a block and an attribute look almost alike and mean different
things:

```hcl
example {
  name = "service"
}

example = {
  name = "service"
}
```

The first is a block named `example`. The second assigns an object value to an
argument named `example`. Parsing settles which one an author wrote; only a
schema settles whether that form is allowed here and what its contents mean.

> **P7 (A) - Schema-Directed Interpretation.** Syntax alone does not determine
> the full meaning of a configuration body. Its schema defines the permitted
> arguments and block types, their value constraints, and their nesting rules.

Core supplies the schemas for its own constructs; providers supply the schemas
for provider configurations and resource types. That split determines how far a
tool can get on its own. Parsing, structural analysis, and reference extraction
all work without a provider schema, which is why an editor can offer navigation
and dependency analysis before `terraform init` has run. Anything
provider-specific — whether an argument exists, what type it takes, whether
changing it forces replacement — waits for the schema. Installing providers is
how the CLI obtains it, not a logical precondition for static analysis.

The representation lives in `internal/configs/configschema`. A `Block` holds
attributes and nested block types. Each attribute carries a type or a nested
object schema plus flags — `Required`, `Optional`, `Computed`, `Sensitive`,
`WriteOnly` — and each nested block type carries a nesting mode: single, group,
list, set, or map. Three derivations do most of the work, and they answer
different questions. `DecoderSpec()` produces an HCL decoding specification and
so governs how configuration is read. `ImpliedType()` produces a cty type and
so governs the shape of the resulting value. `CoerceValue()` converts an
existing value into that shape. [Source: [configuration schemas][schemas].]

### 4.2 Native and JSON Forms

Terraform's JSON syntax is the same language in a different notation, meant
chiefly for generators and tools. Because a JSON object is just an object, the
surrounding Terraform schema decides whether a given object represents a
Terraform value or nested configuration; the notation alone cannot say.

A language feature therefore needs a defined JSON representation as well as a
native one. That does not require identical notation, nor preservation of
comments and formatting. It requires that generated configuration can express
the feature's semantics without falling back on a native-only escape hatch.
[Source: [JSON configuration syntax][json-syntax].]

### 4.3 Restricted Evaluation Contexts

A few expressions have to be evaluated before the resource graph exists at all.
Module installation is the clearest case: Core cannot load a module's
configuration until it knows where that module comes from.

The v1.16 documentation allows module `source` and `version` to use constant
expressions built from local values and from input variables declared
`const = true`:

```hcl
variable "registry" {
  type  = string
  const = true
}

module "app" {
  source  = "${var.registry}/example/app/aws"
  version = "1.2.0"
}
```

This buys a limited amount of early evaluation. What it deliberately does not
buy is a path from a resource result back to the question of which
configuration must be installed first. [Sources: [module blocks][module-doc];
[input variables][variable-doc].]

> **P8 (P) - One Evaluation Semantics.** Restricted evaluation contexts use
> the same expression language. Their restrictions concern available inputs
> and capabilities; they must not silently give ordinary expressions a
> different meaning.

Restricted contexts still differ from one another, and a design has to say how.
A resource reference is simply unavailable during initialization. An
unpredictable function yields an unknown during planning rather than an error.
Those restrictions belong in the specification, stated directly. Sharing the
surface syntax is not by itself an account of how the phases relate.

## 5. The Value Model

Terraform represents values and types with [`cty`][cty]. Beyond ordinary
concrete values the model has nulls, unknowns, refinements, and marks, and all
of them appear both in expression evaluation and in the contracts between Core
and providers.

### 5.1 Types and Conversion

The primitives are `string`, `number`, and `bool`. Collections come in two
families, and the difference between them settles most conversion questions.
Lists, sets, and maps hold elements of one common type. Objects and tuples hold
elements of possibly different types — named for objects, positional for
tuples.

Conversion moves between those families, and it loses information doing so.
`["a", "b"]` is a tuple of two strings, and converting it to `list(string)`
succeeds because the elements already share a type; `["a", 1]` converts too, by
turning `1` into `"1"`. Converting an object to a constraint naming fewer
attributes also succeeds, and the omitted attributes are then simply gone:

```hcl
variable "server" {
  type = object({ name = string })
}
```

Pass `{ name = "web", port = 8080 }` to that variable and `var.server` is
`{ name = "web" }`. Nothing downstream can reach `port`, because the converted
value no longer has it. A module's type constraints define a view of the
caller's value, not merely a check on it.

`any` is a placeholder for a type to be inferred, not a universal type that
holds anything. `list(any)` still demands a single element type, so `["a", 1]`
becomes a `list(string)` rather than a list of mixed values. The implementation
spells this absence of constraint `cty.DynamicPseudoType`, which is a different
thing from a language-level "any value."
[Sources: [type constraints][type-constraints]; cty's `convert` package.]

Numbers have one type, not an integer type and a float type. A provider's API
may well require an integer within a range, and that restriction belongs at the
conversion boundary where the value reaches the provider. The expression
language does not infer a machine type from the way a number was spelled.

### 5.2 Null and Empty Values

Null means absence. `""`, `0`, and `[]` are all present values, and none of
them is null — a distinction that matters whenever a provider has to tell "the
author set this to empty" from "the author said nothing." At an argument
boundary, null normally means no value was supplied, as interpreted by that
context's schema, defaults, and conversion rules.

Two Go-level spellings are easy to confuse. `cty.NullVal(type)` is a real
Terraform value that happens to be null. `cty.NilVal` is a zero value meaning
"no cty value here at all." Returning the second where the first was intended
turns an internal gap into a language-level null.

Null and unknown answer different questions. A known null says the value is
absent, definitively. An unknown says Terraform does not know yet, and null may
be among the possibilities. The asymmetry shows at apply: a planned null that
becomes non-null contradicts the plan, while an unknown that resolves to null
does not. [Source:
[`internal/plans/objchange/compatible.go`][object-compatible].]

On an input variable, `nullable = false` rejects null for the variable itself
and stops there. A variable of type `object({ name = string })` declared
`nullable = false` still accepts `{ name = null }`; only the outer value is
constrained. Defaults interact with this, since a default can replace a null
the caller supplied. [Source: [input variables][variable-doc].]

### 5.3 Unknown Values

Unknowns are what let Terraform plan at all. A subnet configured with the ID of
a VPC that does not exist yet has a perfectly describable desired state, except
for that one value:

```text
+ resource "aws_subnet" "example" {
    + cidr_block = "10.0.1.0/24"
    + vpc_id     = (known after apply)
  }
```

The alternative — inventing a placeholder string and hoping that substituting
the real value later changes nothing else — is precisely what unknowns exist to
avoid. Martin Atkins develops the argument in
[*Unknown Values: The Secret to Terraform Plan*][unknown-values].

An unknown usually carries a known type, and may carry further constraints, but
neither is guaranteed: `cty.DynamicVal` is unknown with no type at all, and an
unknown without a non-null refinement may still turn out to be null.

Knowledge can also be partial within a single value. An object can have known
attribute names and an unknown value for one of them; a tuple can have a known
length and unknown elements. Core and providers must keep an unknown collection
distinct from a known collection of unknown elements, because the second tells
you how many elements there are and the first does not.

#### 5.3.1 Unknowns and the Programming Model

The evaluator handles unknowns on the configuration's behalf. There is no
promise object to await and no handle to inspect; an unknown flows through
expressions as an ordinary value.

> **P9 (A) - Honest Unknown.** Evaluation must not substitute an unsupported
> concrete value for missing information. It may produce a known result only
> when the available information determines that result; otherwise it must
> retain an appropriate unknown or report a justified error.

Both halves of that rule have teeth. A known result can come out of unknown
inputs whenever the inputs settle it: `length(["a", b, c])` is `3` no matter
what `b` and `c` become. And an error can be justified before any value
arrives, because type information is enough — taking `.name` from an unknown
string is invalid, since no string has attributes.

> **P10 (A) - Knownness Is Not Observable.** Configuration cannot branch on
> whether an ordinary value is currently known. The value's availability to
> the runtime is not a separate input to the configuration's meaning.

The temptation P10 rules out is an `is_known` predicate. With one, a
configuration could describe one desired state during planning and a different
one during apply, purely because more information had arrived in between, and
the plan an operator approved would describe something that was never going to
be built. Terraform instead evaluates the same expression again against a
fuller context.

Core itself inspects knownness constantly, and must: to render
`(known after apply)`, to decide which checks can run yet, to diagnose an
unknown instance key, to defer work in runtimes that support deferral. P10
constrains the configuration, not the host. The distinction is about who is
permitted to branch on it. [Source: [*Unknown Values*][unknown-values].]

How much stays unknown is partly a provider's choice. A provider that can
derive an attribute from its inputs may return it as known, and plan precision
improves for everything downstream; "known only after apply" describes one
provider version's implementation rather than a property of the attribute.
The converse error is the dangerous one. An unknown does not mean "unchanged
from prior state," and Core will not assume an equality the provider never
asserted. [Terraform issue #30937][unknown-discussion] covers both the
opportunity and the limits remote systems impose.

#### 5.3.2 Increasing Knowledge

> **P11 (A) - Monotonic Knowledge.** For the same expression and semantic
> context, refining its inputs should produce a compatible refinement of its
> result. A result already established as known must not become a different
> known value merely because previously missing information became available.

Concretely: if planning determines that an output is `"web-prod"`, apply cannot
decide it is `"web-staging"` after learning an unrelated instance ID. Knowledge
accumulates; it is not revised.

"Same semantic context" carries the qualification. Changing the configuration,
changing an external input, or running a different operation all put the second
evaluation in a different context, and P11 says nothing about those. It governs
learning more about one computation.

Precision also need not increase at every step. Learning one attribute of an
object can leave another result wholly unknown, which is consistent with a rule
that demands agreement with earlier information rather than steady progress.
Terraform's plan/apply checks enforce the corresponding contract at the
boundaries that matter (§10.4) rather than proving it of every expression.

cty's [compatibility policy][cty-compatibility] makes a similar-sounding
promise on a different axis: a later library version may return a more precise
result than an earlier one did, provided it stays inside the earlier result's
range. That is a promise across library versions, not within a single run.

#### 5.3.3 Known Shape and Instance Expansion

Expansion asks more of planning than ordinary expressions do. The CLI workflow
has to enumerate instances while planning, so `count` must be known and
`for_each` must have known map keys or known set members. Values may stay
unknown; keys may not:

```hcl
resource "aws_instance" "example" {
  for_each = {
    web = aws_ami.web.id # value unknown - fine
    db  = aws_ami.db.id
  }
  ami = each.value
}
```

This works because Terraform can name `aws_instance.example["web"]` and
`aws_instance.example["db"]` without knowing either AMI. Derive the keys
themselves from a resource result and the plan fails, because there is no way
to write down which instances are changing.

The requirement is stronger here because an instance change needs an address
and an argument value does not. A `for` expression or a `dynamic` block can
carry unknown structure forward to later evaluation precisely because nothing
in it needs a separate address. Core does contain gated deferral machinery that
records incomplete expansion without guessing keys (§8.3), but in ordinary CLI
`for_each` an unresolved instance set is an error.
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

A refinement records what is known about an unknown without claiming a value.
The available constraints are modest — not null, a string prefix, a numeric
bound, a range on collection length — and they pay off in deduction. An unknown
ARN refined with the prefix `arn:aws:s3:::` is still unknown, yet
`startswith(arn, "arn:aws:s3")` already evaluates to `true`, and so does
`arn != null`.

> **P12 (D) - Refinement Soundness.** A refinement must describe a range that
> contains every result still legitimately possible. Additional knowledge may
> narrow that range, but must not exclude a valid eventual result.

Producing a refinement is optional; honoring a committed one is not. An
operation may ignore its inputs' refinements and return a sound but vaguer
result, and some boundaries discard refinements entirely. Both are losses of
internal precision, and neither breaks anything. Contradicting a refinement the
plan has already recorded is a different matter, because that range is part of
what apply is permitted to produce. Separating those two scopes is what lets
refinement support be optional while P5 and P11 stay firm.

Terraform produces refinements in its language functions and checks unknown
ranges in `AssertValueCompatible`. The supported refinement kinds are
documented in [cty's refinement guide][refinements]; the compatibility check is
in [`internal/plans/objchange/compatible.go`][object-compatible].

### 5.5 Marks: Sensitive and Ephemeral

A mark travels with a value and says how the value must be handled. Terraform
uses two: sensitive and ephemeral. Unlike a refinement, a mark narrows nothing
about what the value might be.

> **P13 (A) - Mark Propagation.** Evaluation must preserve the handling
> requirements of marked inputs in derived values unless an operation
> explicitly defines and justifies their removal.

Propagation is deliberately conservative, because a general expression
evaluator has no way to tell whether `substr(var.password, 0, 4)` still
discloses something worth protecting. It assumes it does. Operations that
unmark values in order to compute must restore the marks, or the marked paths,
when they rebuild the result. Stripping a mark is a language operation with a
name — `nonsensitive` — rather than a side effect of conversion.
[Sources: [`internal/lang/marks`][marks];
[sensitive-data handling][sensitive-data].]

**Sensitive** controls disclosure in supported output surfaces. It does not
encrypt anything, and it does not keep the value out of saved plans or state
files, which hold it in the clear. Access to those artifacts has to be
controlled separately; a storage backend may encrypt them, but that protection
comes from the backend and not from the mark.

**Ephemeral** keeps a value out of Terraform's persisted plan and state storage
altogether. Ephemeral variables and resources carry temporary credentials and
other run-scoped inputs into the contexts allowed to consume them. Because
nothing is stored, a later phase that needs such a value must obtain it again —
the saved plan is not a hiding place for it.

> **P14 (P) - Ephemeral Non-Persistence.** Terraform must not persist
> ephemeral values in plan or state storage. A context that contributes
> persistent values must reject ephemeral input unless its contract explicitly
> excludes that input from the persisted representation.

The boundary is Terraform's storage, not the process. A permitted provider
operation can consume an ephemeral value and send it to an API; that is the
point of having them. What the contract governs is where values may come to
rest and how they may flow through the language, and it does not substitute for
the provider's own handling and disclosure controls.

**Write-only attributes** draw the same boundary on the provider side. They
accept ordinary or ephemeral input and appear as null in persisted resource
data, which is why merely removing an ephemeral mark while keeping the value
would not satisfy them. Core's ephemeral helpers validate and strip write-only
values where those boundaries fall.
[Sources: [ephemeral values and variables][variable-doc];
[write-only arguments][write-only-doc];
[`internal/lang/ephemeral`][ephemeral-implementation].]

Sets need particular care here, for a reason rooted in how marks compose. A
set's elements are identified by their values, so a mark on one element cannot
stay attached to that element alone; it is hoisted to the containing set. Mark
one password inside a set of credential objects and the whole set becomes
sensitive. That is tolerable for disclosure and fatal for write-only
attributes, which need per-attribute nulling in the persisted object. The
schema validator therefore rejects write-only attributes inside set blocks and
nested set attributes outright. Set element correlation adds further
constraints (§10.3).
[Source: [`configschema/internal_validate.go`][schema-validation].]

### 5.6 Values at the Plan Boundary

Taken together, these facilities let a plan distinguish commitments that a
simpler value model would run together:

| Representation | Information conveyed |
| --- | --- |
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

An address is how every part of Terraform refers to the same thing. The
configuration declares it, a plan proposes a change to it, state records what
it manages, a diagnostic blames it, and an operator targets it. `internal/addrs`
keeps addresses as structured types with string forms, and the type carries
meaning that the string does not.

### 6.1 Declarations and Instances

The central distinction is between what configuration declares and what
expansion produces. The package draws it twice, once for modules and once for
resources:

| Address type | Meaning |
| --- | --- |
| `Module` | A static path through module calls, such as `module.app.module.db`. |
| `ModuleInstance` | A path that includes instance keys, such as `module.app["blue"]`. |
| `Resource` | A resource mode, type, and name relative to a module. |
| `ConfigResource` | A resource in a static module path, independent of expansion. |
| `AbsResource` | A resource within a particular module instance, before selecting a resource instance. |
| `AbsResourceInstance` | A resource instance within a particular module instance. |

So `module.app["blue"].aws_instance.server` names every `server` in one module
instance, and `module.app["blue"].aws_instance.server[0]` names a single
member. The trap is the singleton: with no `count` or `for_each` the instance
prints no key at all, so `aws_instance.server` could be either kind of address
and the string alone will not say which.

> **P15 (A) - Static/Dynamic Separation.** Configuration declarations and
> their expanded instances are distinct objects. Address-taking interfaces
> must specify whether they identify declarations, collections, or concrete
> instances; a managed remote object's state binding belongs to an instance.

Some operations take more than one kind on purpose — `-target` accepts a whole
resource or a single instance — and that is fine as long as the operation says
what the broader address covers. Accepting both is not the same as treating
them as interchangeable. [Source: [`internal/addrs`][addresses].]

### 6.2 Instance Keys

Three keys cover ordinary configurations. `NoKey` marks a singleton, `IntKey`
comes from `count`, and `StringKey` comes from `for_each` on a resource or
module. That last one takes a map or a set of strings and nothing else, which
is why `for_each` is not general sequence iteration; `dynamic` blocks iterate
under separate rules.

A fourth key exists for deferral. `WildcardKey` prints as `[*]` and stands for
expansion that has not been resolved, giving addresses such as
`module.app.aws_instance.server[*]`. It is not a concrete key and never binds a
managed object. A partial address of this kind describes the unresolved part of
expansion without pretending to enumerate its members.
[Sources: `internal/addrs/instance_key.go`; [`internal/instances`][instances].]

### 6.3 Provider Addresses

Two different things are called provider addresses. A **source address** such
as `registry.terraform.io/hashicorp/aws` identifies an implementation to
install. A **configuration address** identifies a configured use of that
implementation, and so has a module context and possibly an alias.

Inside a module, `LocalProviderConfig` uses the module's own local name and
alias. `AbsProviderConfig` uses the resolved source and module path, and its
module path is static — the parser rejects instance keys in provider
configuration addresses. [Source: `internal/addrs/provider_config.go`.]

That staticness is why a module containing its own provider configurations
cannot be called with `count`, `for_each`, or `depends_on`; provider validation
enforces the restriction. The static address model explains part of this and
not all of it, `depends_on` least of all. Reusable modules should take provider
configurations from their caller through the supported association mechanism
instead (§13.3).
[Sources: [`internal/configs/provider_validation.go`][provider-validation];
[providers within modules][module-providers].]

### 6.4 Address Capabilities

Three interfaces describe what can be done with an address, and each one is a
commitment. `Referenceable` marks a subject that reference resolution supports,
so adding one obliges the evaluation context to resolve it. `Targetable`
supplies the containment relationships targeting needs, so adding one obliges
the design to say what selecting it includes. `UniqueKey` yields comparable
keys for maps and sets. None of the three follows from an address merely being
printable.

Graph vertices are a wider category than addresses. Expansion nodes and close
nodes take part in execution without being anything a configuration can name.
P2 governs the dependencies that configuration evaluation creates; it does not
require every piece of runtime bookkeeping to appear in the configuration
namespace.
[Sources: `internal/addrs/referenceable.go`, `targetable.go`, and `unique_key.go`.]

---

# Part III - Runtime and Persistence

Here the representations become behavior. Core builds a graph for each
operation, expands declarations into instances, drives each instance through a
lifecycle, negotiates with providers over what a change means, and records the
results in state. Destruction closes the loop: removal has its own ordering
requirements and its own demands on how long information must survive.

## 7. Graph Construction and Execution

### 7.1 Operation-Specific Graphs

There is no single Terraform graph. Core builds one per operation by running an
ordered pipeline of transformers, each of which adds vertices, attaches
configuration or state to them, draws edges, or prunes work that is not needed.
The pipelines differ because the operations do: a planning graph has to
discover what changes, while an apply graph executes changes a plan already
describes.

The builders draw on several sources at once:

| Information | Purpose |
| --- | --- |
| Configuration | Declarations, expressions, references, and lifecycle settings |
| Prior state | Existing objects, including objects no longer declared |
| Provider schemas and associations | Interpretation and provider selection |
| Planned changes, during apply | Concrete instance operations to execute |
| Run options | Scope and mode, including targeting and destroy planning |

`PlanGraphBuilder` and `ApplyGraphBuilder` fix the transformation order.
`ReferenceTransformer` draws the evaluation dependencies that come from
references. `AttachDependenciesTransformer` records resource dependency
information for state and lifecycle use. The destroy and create-before-destroy
transformations build the operation-specific ordering that lifecycle semantics
require. [Sources: [`graph_builder_plan.go`][plan-graph],
[`graph_builder_apply.go`][apply-graph], and
[`transform_reference.go`][reference-transform].]

Order matters within these pipelines, because each stage consumes what earlier
stages attached. Anything new added to the graph therefore has to say where its
information becomes available and how the later pruning, targeting, and
validation stages should treat it. Cycles in particular are a property of the
finished graph: an edge drawn by a lifecycle transformer closes a loop just as
surely as one drawn from a reference.

Deleting a resource block shows why the graph cannot come from configuration
alone. The object still exists and still has to be destroyed, so its vertex and
its dependency information come from state instead. Such objects are called
**orphans**, a name for what they lack — a current declaration — rather than
for what they still have, which is a managed identity.

### 7.2 Dependency Direction

Edges mean dependency, not execution order. The walker uses them to decide when
a vertex becomes eligible to run, which makes the two coincide during creation
and diverge during destruction.

Take `B` depending on `A`. Creating both requires `A` first, so edge direction
and execution order agree. Destroying both requires `B` first, and the
dependency between them has not changed. Core gets the right order by building
destroy relationships for the destroy operation rather than by walking the
create graph backwards. The [destruction notes][destroying] state the rule
directly: edges represent dependencies rather than order of operations.

`TransitiveReductionTransformer` then removes edges whose ordering some other
path already implies. Reachability survives, and the direct edge set no longer
contains one edge per original reference. Anything inspecting the graph has to
keep three things apart: which edges are direct, what is reachable, and what
the configuration actually referenced.

### 7.3 Reconstructing the Apply Graph

A plan records changes, not edges. `ApplyGraphBuilder` rebuilds the execution
ordering from those changes together with configuration and state, and
`DiffTransformer` turns each recorded instance change into an operation vertex.

The saved plan archive carries configuration and state alongside the change
records, so it holds everything the rebuild needs. It holds it as inputs rather
than as a serialized graph.

The consequence matters for any feature that has to survive `terraform plan
-out` followed by an apply in a different process. Whatever the planning walk
knew in memory is gone by then. Information apply still needs must either be
written into the artifact or be derivable from what the artifact contains.
[Sources: [`internal/plans/plan.go`][plan-model];
[`internal/plans/planfile`][plan-files]; [apply graph builder][apply-graph].]

### 7.4 Walking and Dynamic Subgraphs

Independent vertices run concurrently, up to a limit that defaults to ten and
moves with `-parallelism`. A dependency constrains readiness and says nothing
about when unrelated operations start relative to one another.

Failure does not stop the walk everywhere. Work downstream of a failed vertex
is skipped, work independent of it continues, and vertices marked
`AlwaysRunVertex` run regardless so that required cleanup still happens. This
is one reason an apply can report an error after having already made real
progress.

Some vertices expand during the walk itself. A `GraphNodeDynamicExpandable`
vertex produces a subgraph once the information it needs arrives, and that
subgraph is validated and walked as part of the same operation. Static
reference analysis comes first; concrete instance graphs appear as the walk
proceeds.
[Sources: [`internal/dag/walk.go`][graph-walker];
[`internal/terraform/graph.go`][terraform-graph].]

## 8. Instance Expansion

### 8.1 Registration and Enumeration

`instances.Expander` is the registry that turns repetition into instances, and
it works in two beats. Once Core has evaluated the repetition expression for
some context, it registers the result — singleton, count, or `for_each` — with
the expander. Afterwards any part of the runtime can ask the expander to
enumerate the instances that follow.

The method names reflect those two beats. `SetModuleCount` and
`SetResourceForEach` register; `ExpandModule` and `ExpandResource` enumerate;
`expansion_mode.go` holds the modes themselves. The ordering between them is a
hard requirement: nothing can be enumerated before it has been registered.
[Source: [`internal/instances`][instances].]

That requirement does not make expansion two global passes over the whole
configuration. It interleaves with the graph walk, because it must — a
resource's `for_each` inside a child module may depend on a value that exists
only once the module instance itself has been established.

### 8.2 Nested Expansion

Every instance of a module shares the same declarations and receives its own
input values, so a resource inside the module can expand differently in each
one. A module called with `for_each` over three regions might produce two
subnets in the first and five in the second, making
`module.net["us-east-1"].aws_subnet.this[0]` and
`module.net["eu-west-1"].aws_subnet.this[4]` both ordinary addresses. The
instance total is the sum across expansions, not a resource count multiplied by
a module count.

Static addresses are what keep this tractable. Core can analyze references
between declarations long before any of those keys exist, then switch to
instance-level addresses wherever an operation needs a concrete subject
(§6.1, §7).

Repetition changes also reach existing objects. Drop a key from a `for_each`
map and the object bound to that key does not disappear; it stays in state,
becomes an orphan, and is planned for destruction. Expansion drives removal and
replacement as much as it drives creation.

### 8.3 Unknown Expansion and Deferral

In the CLI workflow, an unknown `count` or an unknown `for_each` instance set
is an error, because Core cannot write down changes for instances it cannot
name. An unknown attribute inside a known instance poses no such problem and is
routine.

Core also carries gated deferral machinery, used by Stacks. Unknown expansion
modes, partial addresses, and deferred-change records let the runtime keep an
account of work it cannot yet plan completely, instead of guessing or failing.
[Sources: [`internal/instances`][instances]; [`internal/plans`][plans-package];
[Stacks runtime][stacks].]

What deferral buys is a split between **changes this round can execute** and
**work that needs another round**. The second category can include a resource
whose instance set is not yet enumerable, represented by its unresolved scope
rather than by invented instance addresses.

The current plan authorizes none of that deferred work; a later round has to
determine and plan it. Deferral therefore affects the completeness of a round
while leaving value fidelity (P5) and plan authorization (P24) intact. It
leaves P10 intact as well: the runtime reports incomplete work rather than
letting the configuration describe a different desired result because knowledge
was thin.

Anything consuming a partial plan needs an explicit completion model, because
"the executable part of this round succeeded" and "all the desired work is
done" are different statements and usually only one of them is true. Reporting,
approval, persistence, and orchestration each have to preserve the difference.
The design background is in [Terraform issue #30937][unknown-discussion].

## 9. The Resource Instance Lifecycle

### 9.1 Current, Deposed, and Tainted Objects

A resource instance holds one **current** object and any number of **deposed**
ones. A deposed object is a former current object that Terraform still tracks
because something has taken its place but it has not been destroyed yet.
Create-before-destroy replacement needs this representation: for a while, two
real remote objects belong to one resource instance address.

Deposed keys tell those retained objects apart. They are internal, and have
nothing to do with the `count` or `for_each` keys an author chose.
[Source: [`internal/states/resource.go`][state-resource].]

**Tainted** is a status meaning Terraform cannot treat an object as ready and
complete — most often the residue of a create that failed partway through. A
tainted object gets planned for replacement rather than accepted as a
realization of the configuration. State operations can set the status
explicitly, though an operator who simply wants a replacement should reach for
`-replace`, which states the intent as a plan option instead of editing status
first. [Sources: [`internal/states/instance_object.go`][state-object];
[planning behaviors][planning].]

### 9.2 Replacement

Replacement pairs the removal of an old object with the creation of a new one,
in one of two orders:

| Ordering | Consequence |
| --- | --- |
| Delete then create | The old object is removed before the replacement is created. |
| Create then delete | The replacement can become current while the old object remains tracked as deposed. |

Two decisions combine here, and they come from different places. Whether to
replace at all comes from a provider requirement, a taint, a
`replace_triggered_by`, or an operator's `-replace`. Which order to use comes
from `create_before_destroy`, which selects the second ordering when a
replacement is needed and never causes a replacement by itself.
[Sources: [planning behaviors][planning];
[lifecycle reference][lifecycle-doc].]

Create-before-destroy also needs the remote system to allow the two objects to
coexist. Unique names, quotas, and exclusive attachments remain the provider's
and the author's problem. A lifecycle setting cannot make an API accept two
objects where it permits one.

### 9.3 Lifecycle Dependencies

`create_before_destroy` does not stay on the resource that declares it. If a
dependent object has to survive until its replacement exists, then destroying
something it depends on early would defeat the purpose — so Core propagates the
setting to those dependencies and rewrites the operation graph to match.
`ForcedCBDTransformer` does the propagation and `CBDEdgeTransformer` makes the
corresponding edge changes. Lifecycle metadata saved in state preserves the
behavior after an object loses its configuration.
[Sources: [`transform_destroy_cbd.go`][cbd-transform];
[resource destruction notes][destroying].]

The resulting operation interleaves creation, dependent updates, and removal of
deposed objects, which is why a single-resource picture of replacement is only
a starting point. The destruction notes contain worked dependency diagrams for
these combinations and are worth reading before changing lifecycle ordering.

## 10. The Core/Provider Contract

### 10.1 Domain Independence

The division of labor is sharp. Core knows about resource modes, addresses,
values, schemas, and operation contracts. A provider knows what those
operations mean against a particular system: which API calls to make, which
attribute changes force replacement, which two spellings of a value mean the
same thing.

> **P16 (A) - Core Is Domain-Agnostic.** Core's resource behavior is expressed
> through domain-independent contracts. Knowledge specific to an infrastructure
> API belongs in the provider and must reach Core through an explicit protocol.

Domain independence is not ignorance. Core's protocol deliberately talks about
replacement, identity, and deferred work, because those concepts recur in every
domain. What stays on the provider's side is the interpretation of one external
system, and it reaches Core through the protocol rather than through special
cases. [Sources: [architecture overview][architecture];
[provider protocol][protocol].]

### 10.2 Managed-Resource Operations

The provider interface splits a resource's lifecycle into operations with
different inputs and different obligations:

| Operation | Role |
| --- | --- |
| `UpgradeResourceState` | Convert stored data from an older schema to the current schema without treating the conversion as refresh. |
| `ReadResource` | Observe the remote object and report its current state. |
| `ValidateResourceConfig` | Diagnose configuration; do not rewrite it. |
| `PlanResourceChange` | Predict the result of a proposed change within the configuration and prior-state rules. |
| `ApplyResourceChange` | Perform the change and return the resulting known state. |
| `ImportResourceState` | Obtain initial state information for an object to bind to an address. |
| `MoveResourceState` | Perform a supported provider-side state conversion for a move between resource types. |

The table divides responsibilities; it does not fix an RPC sequence. For an
instance that is changing, Core typically calls `PlanResourceChange` once
during planning and again during apply once upstream values are known, then
checks the second result against the first. Replacement, no-op, and destroy
paths differ, so "twice per run" is a useful expectation rather than a contract
to code against.

Validation runs early, which makes its obligation unusual: it has to tolerate
values that are legitimately not known yet. A provider may reject what type
information or known values already prove wrong. It may not reject an optional
value merely for being unknown, since evaluation gets another chance later.

Applied, refreshed, and upgraded state must all be free of unknown values when
the operation succeeds. That is a requirement about representation rather than
about knowledge. A provider that cannot observe some remote fact records null
or its best available value, never an unknown (§11.1).
[Sources: [resource-instance change lifecycle][resource-lifecycle];
[`internal/providers/provider.go`][provider-interface].]

### 10.3 Proposed New State and Collection Correlation

Core does not hand the provider a bare configuration. It first merges
configuration with prior state into a **proposed new object**, and `ProposedNew`
encodes how schema flags govern that merge.

The attribute-level rules are short. A non-computed attribute takes its value
from configuration. A computed attribute left null in configuration starts from
its prior value, on the theory that the remote system already chose something
and probably still means it. The provider can then override that starting point
within the protocol's rules, which is exactly why the proposal is a starting
point and not a promise that the prior value survives.

Nested attributes add a case. When a prior nested value contains non-computed
content, Core can tell that configuration once supplied it, and
`optionalValueNotComputable` uses that fact to let removal of the configuration
actually remove the nested object instead of resurrecting it wholesale. The
rule is specific to nested types; it is not a general rule for every
`Optional+Computed` scalar. [Source:
[`internal/plans/objchange/objchange.go`][object-change].]

The Plugin Framework wraps the provider's own implementation in further ordered
steps: apply defaults, mark computed attributes unknown where configuration is
null and the plan differs, run attribute plan modifiers, then run resource plan
modifiers. All of it happens inside Core's consistency contract. That bounds
what a modifier may do — `UseStateForUnknown` is right when the provider can
justify keeping the old value, and wrong as a general way to make uncertainty
disappear. [Source: [Framework plan modification][framework-planning],
supplied v1.18 documentation.]

Nesting mode also decides how old elements get matched to new ones. Lists match
by position, maps by key, and a single nested object matches recursively. Sets
have none of these, because a set element's identity is its own value. Change
one field of one element and, as far as the set is concerned, an old value
vanished and an unrelated new one appeared:

```hcl
ingress { from_port = 80 }   # prior
ingress { from_port = 8080 } # config - a different element entirely
```

Core therefore falls back on a heuristic. `validPriorFromConfig` asks whether a
prior element could have come from a given configuration element with its
computed fields filled in, and pairs them if so. It works often, and it tells
you less than a key would. Provider schema design has to account for which
fields constitute element identity and which are expected to change. The
limitation is confined to nested collections and says nothing about the
instance keys of resource `for_each`.

### 10.4 Plan Validity and Applied Compatibility

Two checks guard two different boundaries, and confusing them makes provider
bugs hard to diagnose.

**`AssertPlanValid`** runs after the provider plans, comparing its planned
object against configuration and prior state. A non-null configured attribute
must normally survive into the plan with its configured value. The provider may
substitute the prior value when the new spelling means the same thing — Core
can compare values but has no way to judge semantic equivalence, so it takes
the provider's word. Computed attributes get more latitude wherever
configuration leaves them unset, and write-only attributes must be null.

Nested blocks are checked according to nesting mode. Configured list and map
blocks have structural correspondence rules, set blocks permit weaker
correlation for the reasons in §10.3, and computed blocks and unknown `dynamic`
results are handled separately again. The check also rejects an unknown block
element outright, directing providers to make a nested attribute's values
unknown rather than the element itself. These details move between versions and
are better read from the applicable schema and protocol than remembered.
Nested blocks are not independent graph nodes, so none of this follows from
equating block count with resource instance count.
[Source: [`internal/plans/objchange/plan_valid.go`][plan-valid].]

**`AssertObjectCompatible`** runs after apply, comparing the result against the
plan. Known values must be equal. Unknowns may become known within their type
and refinement constraints. Collection structure again governs how the
comparison proceeds, with sets treated differently from indexed or keyed
collections. [Source:
[`internal/plans/objchange/compatible.go`][object-compatible].]

The diagnostics these produce — "Provider produced invalid plan" and "Provider
produced inconsistent result after apply" — report a broken contract rather
than an unusual outcome. Their position also bounds what they prove. The real
cause may lie earlier in evaluation or in an external input that changed, and
an error raised after an RPC returns cannot undo remote work already done.

### 10.5 Provider-Defined Functions

A provider-defined function extends expression evaluation, and that is all it
extends. Its contract requires the same result for the same arguments and no
observable side effects — a requirement Core can spot-check but cannot prove
from the behavior it observes.

The two provider extension points are therefore not interchangeable. A resource
operation is expected to touch an external system, under a lifecycle contract
that gives it planning, state, and recovery. A function is expected to compute
a value. Wrapping an operation in function syntax gains none of that machinery;
it only conceals the absence of it.
[Sources: [provider-defined functions][provider-functions-doc];
[provider protocol][protocol].]

Functions live in their own namespace, `provider::name::function`, which keeps
a new built-in function from colliding with a provider's existing one. §16.3
takes up the general namespace problem.

### 10.6 Legacy Compatibility

Protocol versions 5 and 6 coexist, with clients in `internal/plugin` and
`internal/plugin6`. Version 6 is a complete superset of version 5 and can
replace it entirely once every supported legacy provider has migrated.

Protocol version does not account for all compatibility behavior, though. The
`legacy_type_system` flag carries restricted concessions for providers built on
the original SDK. Those concessions exist to keep older value handling
interoperable, not to excuse new implementations from the contract, and they
bring normalization and validation paths of their own including
`NormalizeObjectFromLegacySDK`. A new schema feature has to account for whether
the participating protocol and SDK can represent it at all.
[Sources: [provider protocol definitions][protocol];
[`internal/plans/objchange/normalize_obj.go`][legacy-normalization].]

For design work the distinction to hold onto is between the modern contract and
the compatibility path that preserves older behavior. Both are real, and
neither should be quietly substituted for the other.

## 11. State

### 11.1 Bindings and Known Values

The most important thing in state is not the attribute values. It is the
binding: this resource instance address manages that remote object. Attribute
values help with planning and refresh and cannot substitute for the binding,
because a remote API can enumerate its objects all day without knowing which
Terraform address, in which configuration, is responsible for any of them.

> **P17 (A) - One Address, One Object.** The management model assumes one
> resource instance address for each managed remote object and at most one
> current object at an instance address. Replacement may temporarily retain
> additional, deposed objects at that instance.

Only half of P17 is enforced. The single current-object slot is a fact about
Core's state representation. Unique ownership is an obligation on whoever
models the system, because Core cannot generally recognize that two addresses
refer to the same remote object — not across independent states, and not even
within one. Nothing detects duplicate management on a configuration's behalf.
[Sources: [state purpose][state-purpose]; [`internal/states`][states].]

> **P23 (A) - State Is Wholly Known.** Persisted state snapshots contain no
> unknown values. In-memory planning representations can carry unknown planned
> values, but those must not be mistaken for persisted observations.

Wholly known is not the same as wholly observed. A provider records what its
schema and API access allow, write-only values are excluded by design, and a
null means what the schema says it means rather than "Core looked and found
nothing there."

The implementation marks the boundary with a status. `ObjectPlanned` holds
transient planning placeholders that may contain unknowns. State encoding
cannot preserve those unknowns — it writes them as null — so evaluation that
needs the real planned value has to consult the plan rather than the encoded
state. That internal accommodation is not license for a provider to return
unknowns in applied state.
[Sources: [`internal/states/instance_object.go`][state-object];
[discussion in #30937][unknown-discussion].]

State carries interpretation and lifecycle metadata alongside the values:
dependencies, create-before-destroy status, schema versions, provider-private
data, and resource identity. Dependencies are recorded as configuration-resource
addresses so that the relationships survive removal of the declaration that
created them. [Source: `internal/states/instance_object_src.go`.]

### 11.2 Snapshots, Lineage, and Interfaces

A state file wraps the snapshot in two pieces of history. `Serial` counts
modifications; `Lineage` identifies which history those modifications belong
to. The pair only works together: serial 12 supersedes serial 11 within one
lineage and means nothing at all against a serial from a different one.

Lineage is opaque, set once, and never updated. Compare it byte for byte and
infer nothing from its spelling. State readers likewise check the format
version and report an unsupported format rather than guessing at it.
[Source: [`internal/states/statefile`][state-files].]

The raw state file and the JSON from `terraform show -json` are different
interfaces with different stability. That external tools read raw state does
not promote its internal fields to public API. Automation should use the
documented interface for its task and honor that interface's format version;
backends and migration tools carry further responsibilities the JSON schema
does not describe.
[Sources: [JSON output format][json-format];
[compatibility promises][compatibility].]

### 11.3 Refresh and Drift

Refresh asks each provider for a new observation of its objects, which puts the
provider in an awkward position: it has to tell a meaningful change from a
meaningless one. An API returning the same JSON policy document with different
whitespace has changed nothing, and reporting it would produce a perpetual
diff. An API returning a genuinely different value has changed something, and
hiding that behind the prior spelling would conceal drift.

Core cannot make that judgment for an arbitrary domain, so `ReadResource`
assigns it to the provider.
[Source: [resource-instance change lifecycle][resource-lifecycle].]

A plan keeps two snapshots so it can answer two questions. `PrevRunState` is
what the last run recorded; `PriorState` is what refresh just observed.
Comparing those two produces the drift report. Comparing `PriorState` against
the desired configuration produces the proposed changes. Plan output shows both
results together, which makes it easy to forget they answer different
questions. [Source: [`internal/plans/plan.go`][plan-model].]

Refreshed observations have to reach downstream evaluation as well. A resource
may need no corrective action while one of its observed attributes has changed,
and an output or expression reading that attribute must see the new value
rather than a stale copy.

Refresh-only mode lets an operator review proposed state and output updates
without planning changes to remote objects. The standalone `terraform refresh`
command is deprecated in favor of the reviewable `terraform apply
-refresh-only` workflow — a change in how state updates get approved, not a
change in when Terraform first started reading remote state during planning.
[Source: [refresh command documentation][refresh-doc].]

### 11.4 Coordination and Partial Failure

State coordinates one operation with the next. It is not a transaction log, and
it cannot roll back a provider's external effects. When an apply half succeeds,
the state it writes has to keep whatever progress Terraform can account for;
discarding that would leave the next plan reasoning from a record already known
to be wrong.

Failures come in kinds that recovery has to tell apart. A remote operation can
fail. A remote operation can succeed while Terraform's account of the result is
incomplete. State that Core holds correctly can fail to save. "Apply failed"
covers all three and distinguishes none of them, which is why it is not enough
to infer the condition of the external system.

Backend locking, where supported and enabled, coordinates operations that share
a state. Saved-plan checks add a second guard, rejecting a plan whose state
history no longer matches the one it was produced against. Neither mechanism
locks the remote world, and neither prevents a second independent state from
managing the same object. Turning locking off or running manual state commands
changes those assumptions and deserves corresponding care.
[Sources: [state locking][state-locking]; [`internal/backend`][backends].]

### 11.5 Resource Identity

Structured resource identity gives a provider an object-typed way to say which
remote object a binding refers to, kept separate from ordinary attributes.
State stores the identity data with an identity schema version, the protocol
carries identity schemas and upgrade operations, and configuration-driven
import can supply an `identity` object instead of a string.

The value shows where an API's identity is genuinely composite — an account, a
region, and a name — and where cramming all three into one import string was
never a good representation. What structured identity does not do is collapse
the distinction between a Terraform address and a remote identity. The state
binding still associates the two, and a richer identity establishes nothing
about globally unique management.
[Sources: [import blocks][import-doc]; [provider protocol][protocol];
`internal/states/instance_object_src.go`.]

## 12. Destruction and Dependency Lifetimes

### 12.1 The Reversal Principle

For a simple dependency, destruction reverses the creation requirement: if `B`
needs `A`, remove `B` before removing `A`. This applies to routine
configuration changes as much as to a full `terraform destroy`.

> **P18 (D) - Destroy Is Reversal.** Resource removal derives its dependency
> constraints from the relationships used to manage the objects. Simple
> destruction reverses their creation order; mixed operations and lifecycle
> settings require the corresponding operation-specific graph transformations.

The name is a mnemonic and not an algorithm. Reversing every edge in the
runtime graph would be wrong, because provider configuration, expansion,
cleanup, replacement, and dependent updates each carry their own lifetime
requirements, and none of them behaves like an ordinary resource creation run
backwards. [Source: [resource destruction notes][destroying].]

### 12.2 Configured and Recorded Dependencies

`DestroyEdgeTransformer` connects destruction to the rest of the planned
operations using two sources: the relationships in the current configuration,
and the dependencies recorded in state. The second source does the heavy
lifting for removed declarations and deposed objects, whose original
configuration may not exist any more.

Provider lifetime is the same problem one level up. A provider configuration
can depend on an object that is itself being destroyed, and Core has to keep
enough information and ordering to configure and use that provider until its
last operation finishes. Create-before-destroy adds further constraints on when
old objects may be released.

This is why removal behavior has to be designed alongside creation and update
behavior. A declaration disappearing is a normal input to Terraform, not an
exceptional condition that excuses forgetting the object's dependencies and
provider association. [Sources:
[`transform_destroy_edge.go`][destroy-transform];
[`transform_destroy_cbd.go`][cbd-transform]; §14.3.]

---

# Part IV - Composition and Evolution

Configurations grow, addresses move, related runtimes reuse the same machinery,
and Terraform has to keep older configurations working throughout. This part
covers the mechanisms for each: modules for composition, refactoring
declarations for address and management transitions, the boundaries separating
Test, Stacks, Query, Actions, and policy from Core, and the compatibility
promises that constrain them all.

## 13. Modules

### 13.1 Scope Without an Independent Transaction

A module is a unit of authorship, distribution, scope, and reuse. Its inputs
and outputs expose chosen values while keeping internal names private. What it
is not, by default, is a separate plan/apply transaction.

> **P19 (P) - Modules Are Namespaces.** Module boundaries organize
> configuration and control visibility. A child module participates in its
> caller's overall Core operation rather than executing as an isolated
> lifecycle unit.

The practical consequence is that module boundaries do not gate execution.
Independent resources in different modules run concurrently, and a module
output becomes available as soon as its own dependencies are satisfied rather
than when the module "finishes." Two modules can therefore pass values to each
other, which looks like a cycle on a module-level diagram and is perfectly
acyclic at the level where the dependencies actually exist.

This is sometimes called flattening the module tree. The phrase describes
evaluation semantics rather than the graph, which does contain module expansion
and close nodes and does use dynamic subgraphs. Nor does a module boundary
imply separate persisted state for the child.
[Sources: [architecture overview][architecture];
[`internal/terraform`][terraform-runtime].]

### 13.2 Inputs, Outputs, and Dependencies

Values cross a module boundary through input variables and output values, and
through nothing else. A caller cannot reach into the child's resource
namespace, and a child cannot name arbitrary objects in its caller. The
expressions that supply inputs and consume outputs create the references, and
the references create the dependencies.

`depends_on` on a module call is a blunter instrument than it looks. It makes
the whole child depend on the subject, delaying work throughout the module
including reads that could have happened much earlier. Where the relevant input
and output references already express the dependency, they give Core something
more precise than a whole-module edge.
[Source: [`depends_on` reference][depends-on-doc].]

The interface is more than a list of names. Type constraints decide what a
module accepts and what survives conversion into it (§5.1), `nullable` decides
whether absence is allowed, and sensitive or ephemeral declarations carry
handling obligations across the boundary. Two modules compose when these agree,
not when their attribute names happen to match.

### 13.3 Provider Association

Provider configurations cross module boundaries through a dedicated mechanism
rather than as ordinary input values. A reusable child module declares the
provider requirements it has; the caller supplies configurations that satisfy
them. Default configurations are inherited automatically, and aliases must be
associated explicitly through the `providers` meta-argument.

Local provider names are interpreted per module, so the association has to
resolve to a compatible provider source even when caller and child use
different local names. Version requirements and configured instances play
different roles here: selecting an implementation is not the same as supplying
credentials, a region, or an aliased configuration.

Root-owned provider configurations are the normal design for reusable modules.
Legacy child modules with their own provider blocks remain supported under the
restrictions in §6.3, including incompatibility with module-call `count`,
`for_each`, and `depends_on`. Provider configurations must also stay available
while any object associated with them still needs operations — removal
included. [Source: [providers within modules][module-providers].]

### 13.4 Custom Conditions

Variable validation, preconditions, postconditions, and `check` blocks all
express assumptions an author wants enforced. Where each one sits determines
what it can see and what its failure stops.

A resource precondition runs after instance expansion and before the resource's
arguments are evaluated. It can therefore read `count.index` and `each`, and it
cannot guard the repetition expression that produced them, since by the time it
runs that decision is already made. A postcondition runs afterward, checks
results, and can stop dependent work. Unknown inputs push a check later, until
enough is known to decide it.
[Sources: [custom conditions][conditions-doc];
[lifecycle reference][lifecycle-doc].]

Severity is part of each contract. A failed precondition or postcondition
blocks the operation. A failed `check` assertion is a warning and gates
nothing. Terraform Test folds both into its own assertion and expected-failure
handling, so a test outcome cannot be read off the CLI's warning severity.
[Sources: [check blocks][check-doc]; [`internal/moduletest/run.go`][test-run].]

These constructs let a module author state assumptions next to the values they
concern. Governance policy answers to a different authority: an assertion the
author can delete from the same file is documentation with teeth, not an
independent approval control.

## 14. Refactoring and Management Transitions

### 14.1 Addresses Change While Objects Remain

Rename a resource and Terraform sees two unrelated events: one declaration
vanished and another appeared. State still binds the existing object to the old
address, so the plan would destroy and recreate. A `moved` block tells Core to
reinterpret the binding before ordinary planning starts:

```hcl
moved {
  from = aws_instance.app
  to   = aws_instance.web
}
```

The object survives the rename. What the move does not promise is that the rest
of the plan is a no-op, since the new configuration may still call for an
update or a replacement. A supported move between resource types goes further
and asks the provider to convert state, which is a provider call. Neither
"a move is free" nor "a move is a destroy" describes the mechanism.
[Sources: [moved blocks][moved-doc]; [`internal/refactoring`][refactoring].]

Three constructs handle management transitions, and they differ in what they do
to the remote object:

| Construct | Management transition |
| --- | --- |
| `moved` | Associate an existing binding with a new configuration address. |
| `removed` | End management of the selected object or objects, with the declared removal behavior. |
| `import` | Establish management of an existing remote object at an address. |

`removed` destroys by default. Adding `lifecycle { destroy = false }` drops the
state binding and leaves the remote object alone — the difference between "this
should not exist" and "this should not be mine." `import` calls the provider
and reads the object rather than assigning arbitrary data to an address.
[Sources: [removed blocks][removed-doc]; [import blocks][import-doc].]

### 14.2 Persistent Declarations, Not Repeated Commands

> **P20 (P) - Refactoring Statements Are Inert.** A refactoring declaration
> records a relationship between configuration and management history. Once
> that transition has been satisfied, retaining the declaration must not by
> itself repeat the completed operation.

"Inert" describes the relationship to history, not an absence of effects.
Processing a declaration can call a provider and change state, and an invalid
or contradictory one produces diagnostics rather than quietly doing nothing.
What it will not do is perform a completed transition a second time.

That property is what lets a module accumulate a chain of `moved` declarations
so that users upgrading from any earlier version arrive at the current address
structure. The declarations become part of the module's migration contract, and
deleting one too early breaks an upgrade path even though the current
configuration is otherwise unchanged.

Core validates move relationships for conflicts and cycles, orders the valid
chains, and applies them to the working state; `move_validate.go` and
`move_execute.go` hold that logic. Because the transitions live in
configuration, they can be reviewed in a pull request and previewed in a plan.
Imperative state commands reach a similar final binding and acquire none of
those properties. [Sources: [`internal/refactoring`][refactoring];
[module refactoring documentation][module-refactoring].]

### 14.3 Metadata Lifetime

Deleting a resource block also deletes the information needed to remove its
object: which provider configuration to use, what it depended on, what
destroy-time configuration applied. The declaration of desired state disappears
well before the remote object does.

> **P21 (P) - Metadata Lifetime.** Information required by an object's
> remaining operations must survive until those operations are complete,
> including when the declaration that originally supplied it has been removed.

Three mechanisms share the work. State preserves dependencies and lifecycle
metadata. Provider configurations have to stay in the configuration until the
objects using them are gone. And a `removed` block holds removal-specific
intent — including `connection` and destroy-time `provisioner` blocks — without
continuing to declare the object as desired. The lifetime argument is developed
in the *removed blocks* design proposal; current behavior is in the
[removed-block reference][removed-doc].

None of this argues for copying every former declaration into state. Some
information must not be persisted at all, ephemeral values above all. The
design question is which information is stored, which is reconstructed, and
which has to be supplied again — and the same question applies to deposed
objects, provider cleanup, and recovery from a partially completed operation.

## 15. Related Languages and Runtimes

Several systems around Terraform share its syntax and its value model, which
makes them look more alike than they are. Shared libraries say nothing about
where evaluation, state, and approval boundaries fall. This section maps the
relationships well enough to reason about a cross-cutting feature; it does not
specify any of these systems.

### 15.1 Terraform Test

Terraform Test adds `run` blocks that plan or apply a configuration and check
assertions against the result, with mocks and overrides to replace selected
provider or module behavior. Each `run` invokes the ordinary Core runtime.
Everything around it — orchestration, evaluation contexts, state management,
result handling — belongs to the test framework.

So "Test uses the same runtime" is half the story. Resource behavior really is
exercised through Core, and setup, assertions, expected failures, and cleanup
are not. A cross-cutting feature has to say which half it lands in, and how a
test can supply inputs to it and observe its outcome.
[Sources: [Terraform tests][tests-doc]; [`internal/moduletest`][module-test].]

### 15.2 Stacks

Stacks is an orchestration layer over trees of Terraform modules. Its address,
configuration, plan, state, and runtime packages play roles analogous to Core's,
and component evaluation calls down into the module runtime underneath.

The key difference from §13 is that a component *is* a planning and state
boundary, which an ordinary child module is not. Stacks coordinates operations
across those components and supports deferred work when a round cannot be
completed. Executable changes still carry a plan contract; orchestration
decides how later rounds finish the rest.
[Source: [`internal/stacks/README.md` and runtime packages][stacks].]

A different boundary permits a different contract. It does not grant an
exemption from having one, because these contracts have consumers too. A rule
about a single module-tree operation transfers to a Stack only once someone has
identified the corresponding scope.

### 15.3 Query

Query uses provider-backed `list` operations to discover existing objects, and
can generate configuration for managing them afterwards. It has its own
configuration decoding and command integration while reusing provider and
planning infrastructure.

Discovery is not the state binding that establishes ownership, and the line
between Query and a data source is not about cardinality — a data source can
return a collection or an empty result too. What separates them is the
contract. `list` is defined around discovery and configuration generation; a
data source returns a provider-defined value.
[Sources: `internal/configs/query_file.go`;
[`internal/command/query.go`][query-command].]

### 15.4 Actions

An `action` block declares a provider-defined operation, and declaring one is
separate from invoking it. Invocation can be requested directly or attached to
a resource lifecycle trigger.

Actions run under their own invocation protocol rather than the persistent
object-management contract that governs a managed resource. They share provider
configuration and repetition mechanisms with resources, which is not enough to
give them resource attributes, refresh, or convergence. An action declaration
should be read through its documented invocation behavior.
[Sources: [action blocks][action-doc]; [invoking actions][invoke-actions-doc];
`internal/configs/action.go`.]

### 15.5 Policy Integration

Governance policy judges whether an operation is acceptable under rules that
can answer to someone other than the module author. Three things define its
relationship to Core: which values and metadata are available to evaluate, when
evaluation happens, and what an enforcement result does to the operation.

At the cited revision, `internal/policy` defines a client with `Setup` and with
resource, provider, and module evaluation requests. Those requests distinguish
current attributes from prior attributes, and carry metadata and redaction
information alongside them.
[Source: [`internal/policy/policy.go`][policy-client].]

That is the Core-side integration and nothing more. Release availability,
policy language semantics, and deployment requirements come from the policy
product's own versioned documentation.

### 15.6 Shared Features, Explicit Boundaries

A new value mark, provider capability, or artifact field can have consumers in
several of these systems at once, and each consumer needs a deliberate answer:
inherit Core's behavior, translate it at a defined boundary, or reject the use
with a diagnostic that explains why.

Identical syntax is not always the right goal, and different syntax is not by
itself a semantic divergence. The comparison worth making is of contracts —
what is evaluated, against which state, what is planned, what is approved, and
what is persisted. Appendix B carries these questions into the interaction
checklist.

## 16. Compatibility

### 16.1 The Documented Promise

Terraform's v1.x compatibility promises cover much of the language, a specified
set of CLI workflows, provider communication, and installation protocols, with
the aim of keeping valid configurations and protected automation working across
v1.x upgrades. Provider behavior is versioned separately and is not part of
Core's promise.

Automation gets JSON output modes and exit status codes as documented
interfaces. Human-readable output and logs are neither, and parsing them means
tracking an unstable surface by choice. Compatibility does not run backwards
either: an older Terraform release is under no obligation to read a newer state
format or understand a newly introduced language feature.
[Source: [v1.x compatibility promises][compatibility].]

The published policy carries important qualifications:

| Area | Qualification |
| --- | --- |
| Validity | The promise generally concerns configurations that can plan and apply without errors. |
| Invalid configuration | Error handling may change; errors can be detected earlier. |
| Implementation bugs | Documented behavior may take precedence, with care to limit compatibility impact. |
| Experiments | Experimental features may change or be removed. |
| Existing deprecations | Explicit deprecation cycles may end during v1.x. |
| Exceptional changes | Critical security issues or changes in external dependencies can justify changes outside the usual expectation. |

Changes to diagnostics are judged by their effect on protected behavior, not by
which phase emits them. The policy explicitly permits changes to
invalid-configuration handling and favors detecting errors earlier.

### 16.2 Supported Behavior Becomes a Contract

> **P22 (P) - Supported Behavior Is a Contract.** Within the documented
> compatibility scope, valid configurations and supported interfaces create
> obligations for later versions. A design must account for the behaviors it
> permits, not only the examples it intends to encourage.

P22 does not freeze every implementation detail. It asks which choices become
*observable*, because those are the ones that turn into commitments. Accepting
several spellings of an argument, exposing an address syntax, permitting a
combination of features — each can create a dependency that nobody notices
until removing it is expensive.

The practical consequence favors restraint. An unsupported combination that
errors clearly leaves room for a well-defined extension later; a supported
combination can be narrowed only with careful attention to compatibility, and
"the implementation would be simpler" is not sufficient reason. Explicit errors
and documented boundaries are the right tools while intended semantics are
still unsettled. This is among the central lessons of the supplied
*Hitchhiker's Guide to Language Design*.

### 16.3 Namespaces

Several Terraform namespaces mix Core-defined names with names that modules and
providers supply. A new resource meta-argument competes with every provider's
existing arguments. A new module meta-argument competes with every input
variable anyone has declared. Predefined expression roots raise the same issue
against resource type names.

A word unused in Core is not a word unused in the ecosystem, which makes
reserving one a compatibility decision before it is a design decision. The
discussion of `enabled` in [#21953][conditional-resources] works through the
module and provider exposure concretely, independently of whether the feature
itself was a good idea.

Separate namespaces reduce the exposure. Provider-defined functions live under
`provider::name::function`, so adding a built-in function cannot collide with a
provider's. The broader namespace analysis appears in the *Language Editions*
proposal.

### 16.4 Editions and Versioned Semantics

The *Language Editions* proposal explores selecting language semantics per
module, so that different editions could coexist in one configuration. That is
design background rather than an upgrade mechanism available to practitioners:
at the cited revision, the parser recognizes only the default `TF2021` edition.
[Source: [`internal/configs/experiments.go`][editions-parser].]

A future edition mechanism would have to define what changes at a module
boundary and what stays shared — values, provider association, addresses,
artifacts. A syntax selector alone does not answer those cross-version
questions. Stacks defines a different orchestration boundary again, and has to
be evaluated against its own compatibility commitments.

### 16.5 Evolution of the Model

Terraform has added a great deal while keeping three things fixed: the
plan/apply model, typed partial values, and persistent management bindings.
Those foundations explain how new features fit alongside old ones. They do not
establish that the current abstractions are the only workable ones.

What an architectural reference can do is make assumptions explicit enough to
argue with. A design that preserves them inherits the existing contracts as
explanation. A design that changes them takes on the work of identifying the
affected consumers and establishing the replacement contract. That is what the
numbered principles and the dependency table are for — not to presume that an
earlier decision must stand.

---

# Appendices

## Appendix A - Principle Index and Architectural Dependencies

The table summarizes the numbered principles. Full statements and their
qualifications are in the sections cited. **A**, **D**, and **P** are
architectural axiom, derived guarantee, and design policy, as defined in §0.2;
they do not rank how serious a violation would be.

**Dependencies in this model** names the supporting assumptions used by the
account given here, and nothing stronger. It is not a necessary-and-sufficient
relation, not an exhaustive list of implementation dependencies, and not a proof
that replacing an assumption makes the supported property impossible. A blank
entry means this table identifies no other numbered principle as support.

| ID | Kind | Name | Summary | Section | Dependencies in this model |
| --- | --- | --- | --- | --- | --- |
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

Reading a row: P18 cites P21 because destruction needs dependency information
that outlives the configuration. P14 cites P13 because propagating the ephemeral
mark is how its storage boundary gets enforced. P6's row is deliberately
incomplete — convergence also requires stable inputs, adequate scope, and
conforming provider behavior, none of which is a numbered principle.

Use the table to trace a design question into the sections that discuss it. It
will not tell you *why* a particular proposal affects a particular contract;
that argument still has to be made.

## Appendix B - Feature Interaction Checklist

These are the architectural questions that come up again and again. The
checklist organizes an investigation; it does not certify a design by having
every row ticked.

| Area | What the design should establish |
| --- | --- |
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

Evidence means implementation paths and representative interactions, especially
interactions that cross phases. A passing test of an isolated, successful create
path establishes nothing about the saved-plan contract, about removal, or about
whether sensitive and ephemeral handling survives.

## Appendix C - Scope Notes

The principle names are short, which makes them easy to cite and easy to
over-read. These are the boundaries that matter most, collected in one place.

| Boundary | Scope |
| --- | --- |
| Expression computation and graph operations | P3 does not claim that planning has no external interactions. Refresh, data-source reads, and ephemeral-resource operations are distinct from expression computation. |
| Partial operations | Targeting and deferral limit what a round establishes about the whole configuration. They do not themselves authorize unplanned changes or contradict the executable plan's known values. |
| External inputs | Filesystem contents and other changing inputs need a lifetime model. P11 concerns refined knowledge of the same computation, not arbitrary reevaluation against a changed world. |
| Provisioners | Provisioners run configured commands under their documented lifecycle rules. Their command effects are not modeled as ordinary provider resource attributes, and resource value checks are not a specification of those effects. |
| Legacy providers | Compatibility paths preserve particular older behaviors. They are not the default contract for new provider implementations. |
| Related languages | The principles apply at identified boundaries. A Stack component operation and a complete Stack are not the same scope; neither is a test run and its surrounding test orchestration. |

Provisioner behavior is documented in the
[provisioner reference][provisioners-doc].
The other boundaries are developed in the sections cited by their principle
numbers.

## Appendix D - Limits of This Reference

This paper is not an operational specification for backends, provider SDKs, or
the related language family. Three areas in particular need more detailed
references before implementation work.

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
| --- | --- |
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
cover evaluation and handling metadata.
[`internal/plans/objchange`][object-change-package] contains proposed-state
construction and consistency checks.

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
