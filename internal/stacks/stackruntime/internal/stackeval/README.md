# Terraform Stacks Runtime Internal Architecture

This directory contains the guts of the Terraform Stacks language runtime.
The public API to this is in the package two levels above this one,
called `stackruntime`.

The following documentation is aimed at future maintainers of the code in
this package. There is no end-user documentation here.

## Overview

If you're arriving here familiar with the runtime of the traditional Terraform
language used for modules -- which we'll call the "modules runtime" in the
remainder of this document -- you will find that things work quite differently
in here.

The modules runtime works by first explicitly building a dependency graph and
then performing a concurrent walk of that graph, visiting each node and asking
it to "evaluate" itself. "Evaluate" could mean something as simple as just
tweaking some in-memory data, or it could involve a time-consuming call to
a provider plugin. The nodes all collaborate via a shared mutable data structure
called `EvalContext`, which nodes use both to read from and modify the state,
plan, and other relavant metadata during evaluation.

The stacks runtime is solving broadly the same problem -- scheduling the
execution of various calculations and side-effects into an appropriate order --
but does so in a different way that relies on an _implicit_ data flow graph
constructed dynamically during evaluation.

The evaluator does still have a sort of global "god object" that everything
belongs to, which is an instance of type `Main`. However, in this runtime
that object is the entry point to a tree of other objects that each encapsulate
the data only for a particular concept within the language, with data flowing
between them using method calls and return values.

## Config objects vs. Dynamic Objects

There are various pairs of types in this package that represent a static object
in the configuration and dynamic instances of that object respectively.

For example, `InputVariableConfig` directly represents a `variable` block
from a `.tfstack.hcl` file, while `InputVariable` represents the possibly-many
dynamic instances of that object that can be caused by being within a stack
that was called using `for_each`.

In general, the static types are responsible for "static validation"-type tasks,
such as checking whether expressions refer to instances of other configuration
objects where the configuration object itself doesn't even exist, let alone
any instances of it. The goal is to perform as many checks as possible as
static checks, because that allows us to give feedback about detected problems
as early as possible (during the validation phase), and also avoids redundantly
reporting errors for these problems multiple times when there are multiple
instances of the same problematic object.

Dynamic types are therefore responsible for everything that needs to respond
to dynamic expression evaluation, and anything which involves interacting with
external systems. For example, creating a plan for a component must be dynamic
because it involves asking providers to perform planning operations that might
contact external servers over the network, and then anything which makes use
of the results from planning is itself a dynamic operation, transitively.

## Calls vs. Instances

A subset of the object types in this package have an additional distinction
aside from Config vs. Dynamic.

`StackCall`, `Component`, and `Provider` all represent dynamic instances
of objects in the configuration that can themselves produce dynamic child
objects. `StackCallInstance`, `ComponentInstance`, and `ProviderInstance`
represent those specific instances.

What all of these types have in common is that the configuration constructs
they represent each support a `for_each` argument for dynamically declaring
zero or more instances of the object.

The breakdown of responsibilities for this process has three parts. We'll
use components for the sake of example here, but the same breakdown applies
to stack calls and provider configurations too:

* `ComponentConfig` represents the actual `component` block in the configuration,
  and is responsible for verifying that the component declaration is even valid
  regardless of any dynamic information.
* `Component` represents a dynamic instance of one of those `component` blocks,
  in the context of a particular stack. This deals with the situation where
  a component is itself inside a child stack that was called using a `stack`
  block which had `for_each` set, and therefore there are multiple instances
  of this component block even before we deal with the component block's _own_
  `for_each` argument.

    The `Component` type is responsible for evaluating the `for_each` expression.

    The `Component` type is also responsible for producing the value that
    would be placed in scope to handle a reference like `component.foo`,
    which it does by collecting up the results from each instance implied
    by the `for_each` expression and returning them as a mapping.
* `ComponentInstance` represents just one of the instances produced by the
  `component` block's own `for_each` expression.

    This type is therefore responsible for evaluating any of the arguments
    that are permitted to refer to `each.key` or `each.value` and could
    therefore vary between instances. It's also responsible for the main
    dynamic behavior of components, which is creating plans, applying them,
    and reporting their results.

## Object Singletons

Almost everything reachable from a `Main` object must be treated as a singleton,
because these objects contain the tracking information for asynchronous
work in progress and the results of previously-completed asynchronous work.

The guarantee of ensuring that each object is indeed treated as a singleton
is the responsiblity of some other object which we consider the child to be
contained within.

For example, the `Main` object itself is responsible for instantiating the
`Stack` object representing the main stack (aka the "root" stack) and then
remembering it so it can return the same object on future requests. However,
any child stacks are tracked inside the state of the root stack object,
and so the root stack is responsible for ensuring the uniqueness of those
across multiple calls. This continues down the tree, with every object
except the `Main` object being the responsibility of exactly one managing
parent.

Failing to preserve this guarantee would cause duplicate work and potentially
inconsistent results, assuming that the work in question does not behave as a
pure function. To help future maintainers preserve the guarantee, there is
a convention that new instances of all of the model types in this package
are produced using an unexported function, such as `newStack`, and that
each of those functions must be called only from one other place within
the managing parent of each type.

(Stacks themselves are a slight exception to this rule because the managing
parent of the main stack is `Main` while the managing parent of all other
stacks is the parent `Stack`. There must therefore be two callsites for
`newStack`, but they are written in such a way as to avoid trampling on each
other's responsibilities.)

The actual singleton objects are retained in an unexported map inside the
managing parent. They are typically created only on first request from
some other caller, via a method of the managing parent. The resulting new object
is then saved in the map to be returned on future calls.

Instances of the `...Config` types should typically be singleton per
`Main` object, because they are static by definition.

Instances of dynamic types are actually only singleton per _evaluation phase_,
since e.g. the behavior of a `ComponentInstance` is different when we're trying
to create a plan than when we are trying to apply a plan previously created.
More on that in the next section.

## Evaluation Phases

Each `Main` object is typically instantiated for only one evaluation phase,
which from the external caller's perspective is controlled by which of the
factory functions they call.

Internally we track evaluation phases as instances of `EvalPhase`, which
is a comparable type that we use internally to differentiate between the
singletons created for one phase and the singletons created for another.

Since currently each `Main` has only one evaluation phase, this is actually
technically redundant: a `Main` instantiated for planning would produce
only objects for the `PlanPhase` phase.

However, the implementation nonetheless tracks a separate pool of singletons
per phase and requires any operation that performs expression evaluation to
explicitly say which evaluation phase it's for, as some insurance both against
bugs that might otherwise be quite hard to track down and against possible
future needs that might call for us needing to blend work for multiple phases
into the same `Main` object for some reason.

* `NewForValidating` returns a `Main` for `ValidatePhase`, which is capable
  only of static validation and will fail any dynamic evaluation work.
* `NewForPlanning` returns a `Main` for `PlanPhase`, bound to a particular
  prior state and planning options.
* `NewForApplying` returns a `Main` for `ApplyPhase`, bound to a particular
  stack plan, which itself includes the usual stuff like the prior state,
  the planned changes, input variable values that were specified during
  planning, etc.
* `NewForInspecting` returns a `Main` for `InspectPhase`, which is a special
  phase that is intended for implementing less-commonly-used utilities such
  as something equivalent to `terraform console` but for Stacks. In this
  case, the evaluator is bound only to a prior state, and just returns values
  directly from that state without trying to plan or apply any changes.

    This phase is also handy for unit testing of parts of the runtime that
    don't rely on external side-effects; many of the unit tests in this
    package do their work in `InspectPhase`, particularly if testing an
    object whose direct behavior does not vary based on the evaluation
    phase. It's still important to test in other phases for operations whose
    behavior varies by phase, of course!

## Expression Evaluation

The most important cross-cutting behavior in the language runtime is the
evaluation of user-provided expressions. The main function for that is
`EvalExpr`, but there's also `EvalBody` for evaluating all of the expressions
in a dynamic body at once, and extensions such as `EvalExprAndEvalContext`
which also returns some of the information that was used during evaluation
so that callers can produce more helpful diagnostic messages.

The actual evaluation process involves two important concepts:

- `EvaluationScope` is an interface implemented by objects that can have
  expressions evaluated inside them. Each `Stack` is effectively a
  "global scope", and then some child objects like `Component`, `StackCall`,
  and `Provider` act as _child_ scopes which extend the global scope with
  local context like `each.key`, `each.value`, and `self`.

    An evaluation scope's responsibility is to translate a `stackaddrs.Reference`
    (a representation of an already-decoded reference expression) into an object
    that implements `Referenceable`.

- `Referenceable` is an interface implemented by objects that can be referred
  to in expressions. For example, a reference expression like `var.foo`
  should refer to an `InputVariable` object, and so `InputVariable` implements
  `Referenceable` to decide the actual value to use for that reference.

    The responsibility of an implementation of this interface is simply to
    return a `cty.Value` to insert into the expression scope for a particular
    `EvalPhase`. For example, a `Component` object implements this interface
    by returning an object containing all of the output values from the
    component's plan when asked for `PlanPhase`, but returns the output values
    from the final state instead when asked for `ApplyPhase`.

Overall then, the expression evaluation process has the following main steps:

1. Analyze the expression or collection of expressions to find all of the
   HCL symbol references (`hcl.Traversal` values).
2. Use `stackaddrs.ParseReference` to try to raise the reference into one of
   the higher-level address types, wrapped in a `stackaddrs.Reference`.
   
   We fail at this step for syntactically-invalid references, but this step
   has no access to the dynamic symbol table so it cannot catch references to
   objects that don't exist.
3. Pass the `stackaddrs.Reference` value to the caller's selected
   `EvaluationScope` implementation, which checks whether the address refers
   to an object that's actually declared, and if so returns that object.
   This uses `EvaluationScope.ResolveExpressionReference`.

    This step fails if the reference is syntactically valid but refers to
    something that isn't actually declared.

    Objects that expressions can refer to must implement `Referenceable`.
4. Call `ExprReferenceValue` on each of the collected `Referenceable` objects,
   passing the caller's `EvalPhase`.

    That method must then return a `cty.Value`. If something has gone wrong
    upstream that prevents returning a concrete value, the method should return
    some kind of unknown value -- ideally with a type constraint, but as
    `cty.DynamicVal` as a last resort -- so that evaluation can continue
    downstream just enough to let the call stacks all unwind and collect
    all the error diagnostics up at the top.
5. Assemble all of the collected values into a suitably-shaped `hcl.EvalContext`,
   attach the usual repertiore of available functions, and finally ask the
   original expression to evaluate itself in that evaluation context.

    Failures can occur here if the expression itself is invalid in some way,
    such as trying to add together values that cannot convert to number, or
    other similar kinds of type/value expectation mismatch.

## Checked vs. Unchecked Results

Data flow between objects in a particular evaluator happens mostly on request.

For example, if a `component` block contains a reference to `var.foo` then
as part of evaluating that expression the `Component` or `ComponentInstance`
object will (indirectly, through the expression evaluator) ask the
`InputVariable` object for `variable "foo"` to produce its value, and only
at that point will the `InputVariable` object begin the work of evaluating
that value, which could involve evaluating yet another expression, and so on.

Because the flow of requests between objects is dynamic, and because many
different requesters can potentially ask for the same result via different
call paths, if an error or warning diagnostic is returned we need to make sure
_that_ propagates by only one return path to avoid returning the same
diagnostic message multiple times.

To deal with that problem, operations that can return diagnostics are typically
split into two methods. One of them has a `Check` prefix, indicating that
it is responsible for propagating any diagnostics, and the other lacks the
prefix.

For example, `InputVariable` has both `Value` and `CheckValue`. The latter
returns `(cty.Value, tfdiags.Diagnostics)`, while the former just wraps the
latter and discards the diagnostics completely.

This strategy assumes two important invariants:
- Every fallible operation can produce some kind of inert placeholder result
  when it fails, which we can use to unwind everything else that's depending
  on the result without producing any new errors. (or, in some cases, producing
  a minimal amount of additional errors that each add more information than
  the original one did, as a last resort when the ideal isn't possible).
- Only one codepath is responsible for calling the `Check...` variant of the
  function, and everything else will use the unprefixed version and just
  deal with getting a placeholder result sometimes.

This is quite different than how we've dealt with diagnostics in other parts
of Terraform, and does unfortunately require some additional care under future
maintenence to preserve those invariants, but following the naming convention
across all of the object types will hopefully make these special rules easier
to learn and then maintain under future changes.

In practice, the one codepath that calls the `Check...` variants is the
"walk" codepath, which is discussed in the next section.

## Static and Dynamic "Walks"

As discussed in the previous section, most results in the stacks runtime
are produced only when requested. That means that if no other object in
the configuration were to include an expression referring to `var.foo`,
it might never get any opportunity to evaluate itself and raise any errors
in its declaration or definition.

To make sure that every relevant object gets visited at least once, each of
the main evaluation phases (not `InspectPhase`) has at least one "walk"
associated with it, which navigates the entire tree of relevant objects
accessible from the `Main` object and calls a phase-specific method on
each one.

There are two "walk drivers" that arrange for traversing different subsets
of the objects:
- The "static" walk is used for both `ValidatePhase` and `PlanPhase`, and
  visits only the objects of `Config`-suffixed types, representing static
  configuration objects.
- The "dynamic" walk is used for both `PlanPhase` and `ApplyPhase`, and
  visits both the main dynamic objects (the ones of types with no special
  suffix) and the objects of `Instance`-suffixed types that represent
  dynamic instances of each configuration object.

The "walk driver" decides which objects need to be visited, calling a callback
function for each object. Each phase calls a different method of each visited
object in its callback:
- `ValidatePhase` calls the `Validate` method of interface `Validatable`,
  which is only allowed to return diagnostics and should not have any
  externally-visible side-effects.
- `PlanPhase` calls the `PlanChanges` method of interface `Plannable`,
  which can return an arbitrary number of "planned change" objects that
  should be returned to the caller to contribute to the plan, and an arbitrary
  number of diagnostics.
- `ApplyPhase` calls the `CheckApply` method of interface `Applyable`,
  which is responsible for collecting the results of apply actions that are
  actually scheduled elsewhere, since the runtime wants a little more control
  over the execution of the side-effect heavy apply actions. This returns an
  arbitrary number of "applied change" objects that each represents a
  mutation of the state, and an arbitrary number of diagnostics.

Those who are familiar with Terraform's modules runtime might find this
"walk" idea roughly analogous to the process of building a graph and then
walking it concurrently while preserving dependencies. The stack runtime
walks are different in that they are instead walking the _tree_ of objects
accessible from `Main`, and they don't need to be concerned about ordering
because the dynamic data flow between the different objects -- where a method
of one object can block on the completion of a method of another -- causes a
suitable evaluation order automatically.

The scheduling here is dynamic and emerges automatically from the control
flow. The runtime achieves this by having any operation that depends on
expensive or side-effect-ish work from another object pass the data using
the promises and tasks model implemented by
[package `promising`](../../../../promising/README.md).

## Apply-phase Scheduling

During the validation and planning operations the order of work is driven
entirely by the dynamically-constructed data flow graph that gets assembled
automatically based on control flow between the different functions in this
package. That works under the assumption that those phases should not be
modifying anything outside of Terraform itself and so our only concern is
ensuring that data is available at the appropriate time for other functions
that will make use of it.

However, the apply phase deals with externally-visible side-effects whose
relative ordering is very important. For example, in some remote APIs an
attempt to destroy one object before destroying another object that depends
on it will either fail with an error or hang until a timeout is reached, and
so it's crucially important that Terraform directly consider the sequence
of operations to make sure that situation cannot possibly arise, even if
the relationship is not implied naturally by data flow.

We deal with those additional requirements with both an additional scheduling
primitive -- function `ChangeExec` -- and with some explicit dependency data
gathered during the planning phase.

In practice, it's only _components_ that represent operations with explicit
ordering constraints, because nothing else in the stacks runtime directly
interacts with Terraform's resource instance change lifecycle. Therefore
we can achieve a correct result with only a graph of dependencies between
components, without considering any other objects. Interface `Applyable`
includes the method `RequiredComponents`, which must return a set of all
of the components that a particular applyable object depends on.

In practice, most of our implementations of `Applyable.RequiredComponents`
wrap a single implementation that works in terms of interface `Referrer`, which
works at a lower level of abstraction that deals only in HCL-level expression
references, regardless of what object types they refer to. The shared
implementation then raises the graph of references into a graph of components
by essentially removing the non-component nodes while preserving the
edges between them.

Once the plan phase has derived the relationships between components, it
includes that information as part of the plan, so that it's immediately ready
to use in the apply phase without any further graph construction.

The apply phase then uses the `ChangeExec` function to actually schedule the
changes. That function's own documentation contains more documentation about
its usage, but at a high level it wraps the concepts from
[package `promising`](../../../../promising/README.md) in such a way that
it can oversee the execution of each of the individual component instance apply
phases, and capture the results in a central place for downstream work to
refer to. Each component instance is represented by a single task which blocks
on the completion of the promise of each component it depends on, thus explicitly
ensuring that the component instance changes get applied in the correct
order relative to one another.

Since the `ChangeExec` usage is concerned only with component instances, the
apply phase still performs a concurrent dynamic walk as described in the
previous section to ensure that all other objects in the configuration will be
visited and have a chance to announce any problems they detect. The significant
difference for the apply phase is that anything which refers to a component
instance will block until the `ChangeExec`-managed apply phase for that
component instance has completed. Otherwise, the usual data-flow-driven
scheduling decides on the evaluation order for all other object types.
