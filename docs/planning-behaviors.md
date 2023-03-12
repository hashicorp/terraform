# Planning Behaviors

A key design tenet for Terraform is that any actions with externally-visible
side-effects should be carried out via the standard process of creating a
plan and then applying it. Any new features should typically fit within this
model.

There are also some historical exceptions to this rule, which we hope to
supplement with plan-and-apply-based equivalents over time.

This document describes the default planning behavior of Terraform in the
absence of any special instructions, and also describes the three main
design approaches we can choose from when modelling non-default behaviors that
require additional information from outside of Terraform Core.

This document focuses primarily on actions relating to _resource instances_,
because that is Terraform's main concern. However, these design principles can
potentially generalize to other externally-visible objects, if we can describe
their behaviors in a way comparable to the resource instance behaviors.

This is developer-oriented documentation rather than user-oriented
documentation. See
[the main Terraform documentation](https://www.terraform.io/docs) for
information on existing planning behaviors and other behaviors as viewed from
an end-user perspective.

## Default Planning Behavior

When given no explicit information to the contrary, Terraform Core will
automatically propose taking the following actions in the appropriate
situations:

- **Create**, if either of the following are true:
  - There is a `resource` block in the configuration that has no corresponding
    managed resource in the prior state.
  - There is a `resource` block in the configuration that is recorded in the
    prior state but whose `count` or `for_each` argument (or lack thereof)
    describes an instance key that is not tracked in the prior state.
- **Delete**, if either of the following are true:
  - There is a managed resource tracked in the prior state which has no
    corresponding `resource` block in the configuration.
  - There is a managed resource tracked in the prior state which has a
    corresponding `resource` block in the configuration _but_ its `count`
    or `for_each` argument (or lack thereof) lacks an instance key that is
    tracked in the prior state.
- **Update**, if there is a corresponding resource instance both declared in the
  configuration (in a `resource` block) and recorded in the prior state
  (unless it's marked as "tainted") but there are differences between the prior
  state and the configuration which the corresponding provider doesn't
  explicitly classify as just being normalization.
- **Replace**, if there is a corresponding resource instance both declared in
  the configuration (in a `resource` block) and recorded in the prior state
  _marked as "tainted"_. The special "tainted" status means that the process
  of creating the object failed partway through and so the existing object does
  not necessarily match the configuration, so Terraform plans to replace it
  in order to ensure that the resulting object is complete.
- **Read**, if there is a `data` block in the configuration.
  - If possible, Terraform will eagerly perform this action during the planning
    phase, rather than waiting until the apply phase.
  - If the configuration contains at least one unknown value, or if the
    data resource directly depends on a managed resource that has any change
    proposed elsewhere in the plan, Terraform will instead delay this action
    to the apply phase so that it can react to the completion of modification
    actions on other objects.
- **No-op**, to explicitly represent that Terraform considered a particular
  resource instance but concluded that no action was required.

The **Replace** action described above is really a sort of "meta-action", which
Terraform expands into separate **Create** and **Delete** operations. There are
two possible orderings, and the first one is the default planning behavior
unless overridden by a special planning behavior as described later. The
two possible lowerings of **Replace** are:
1. **Delete** then **Create**: first delete the existing object bound to an
  instance, and then create a new object at the same address based on the
  current configuration.
2. **Create** then **Delete**: mark the existing object bound to an instance as
  "deposed" (still exists but not current), create a new current object at the
  same address based on the current configuration, and then delete the deposed
  object.

## Special Planning Behaviors

For the sake of this document, a "special" planning behavior is one where
Terraform Core will select a different action than the defaults above,
based on explicit instructions given either by a module author, an operator,
or a provider.

There are broadly three different design patterns for special planning
behaviors, and so each "special" use-case will typically be met by one or more
of the following depending on which stakeholder is activating the behavior:

- [Configuration-driven Behaviors](#configuration-driven-behaviors) are
  activated by additional annotations given in the source code of a module.

    This design pattern is good for situations where the behavior relates to
    a particular module and so should be activated for anyone using that
    module. These behaviors are therefore specified by the module author, such
    that any caller of the module will automatically benefit with no additional
    work.
- [Provider-driven Behaviors](#provider-driven-behaviors) are activated by
  optional fields in a provider's response when asked to help plan one of the
  default actions given above.

    This design pattern is good for situations where the behavior relates to
    the behavior of the remote system that a provider is wrapping, and so from
    the perspective of a user of the provider the behavior should appear
    "automatic".

    Because these special behaviors are activated by values in the provider's
    response to the planning request from Terraform Core, behaviors of this
    sort will typically represent "tweaks" to or variants of the default
    planning behaviors, rather than entirely different behaviors.
- [Single-run Behaviors](#single-run-behaviors) are activated by explicitly
  setting additional "plan options" when calling Terraform Core's plan
  operation.

    This design pattern is good for situations where the direct operator of
    Terraform needs to do something exceptional or one-off, such as when the
    configuration is correct but the real system has become degraded or damaged
    in a way that Terraform cannot automatically understand.

    However, this design pattern has the disadvantage that each new single-run
    behavior type requires custom work in every wrapping UI or automaton around
    Terraform Core, in order provide the user of that wrapper some way
    to directly activate the special option, or to offer an "escape hatch" to
    use Terraform CLI directly and bypass the wrapping automation for a
    particular change.

We've also encountered use-cases that seem to call for a hybrid between these
different patterns. For example, a configuration construct might cause Terraform
Core to _invite_ a provider to activate a special behavior, but let the
provider make the final call about whether to do it. Or conversely, a provider
might advertise the possibility of a special behavior but require the user to
specify something in the configuration to activate it. The above are just
broad categories to help us think through potential designs; some problems
will require more creative combinations of these patterns than others.

### Configuration-driven Behaviors

Within the space of configuration-driven behaviors, we've encountered two
main sub-categories:
- Resource-specific behaviors, whose effect is scoped to a particular resource.
  The configuration for these often lives inside the `resource` or `data`
  block that declares the resource.
- Global behaviors, whose effect can span across more than one resource and
  sometimes between resources in different modules. The configuration for
  these often lives in a separate location in a module, such as a separate
  top-level block which refers to other resources using the typical address
  syntax.

The following is a non-exhaustive list of existing examples of
configuration-driven behaviors, selected to illustrate some different variations
that might be useful inspiration for new designs:

- The `ignore_changes` argument inside `resource` block `lifecycle` blocks
  tells Terraform that if there is an existing object bound to a particular
  resource instance address then Terraform should ignore the configured value
  for a particular argument and use the corresponding value from the prior
  state instead.

    This can therefore potentially cause what would've been an **Update** to be
    a **No-op** instead.
- The `replace_triggered_by` argument inside `resource` block `lifecycle`
  blocks can use a proposed change elsewhere in a module to force Terraform
  to propose one of the two **Replace** variants for a particular resource.
- The `create_before_destroy` argument inside `resource` block `lifecycle`
  blocks only takes effect if a particular resource instance has a proposed
  **Replace** action. If not set or set to `false`, Terraform will decompose
  it to **Destroy** then **Create**, but if set to `true` Terraform will use
  the inverted ordering.

    Because Terraform Core will never select a **Replace** action automatically
    by itself, this is an example of a hybrid design where the config-driven
    `create_before_destroy` combines with any other behavior (config-driven or
    otherwise) that might cause **Replace** to customize exactly what that
    **Replace** will mean.
- Top-level `moved` blocks in a module activate a special behavior during the
  planning phase, where Terraform will first try to change the bindings of
  existing objects in the prior state to attach to new addresses before running
  the normal planning process. This therefore allows a module author to
  document certain kinds of refactoring so that Terraform can update the
  state automatically once users upgrade to a new version of the module.

    This special behavior is interesting because it doesn't _directly_ change
    what actions Terraform will propose, but instead it adds an extra
    preparation step before the typical planning process which changes the
    addresses that the planning process will consider. It can therefore
    _indirectly_ cause different proposed actions for affected resource
    instances, such as transforming what by default might've been a **Delete**
    of one instance and a **Create** of another into just a **No-op** or
    **Update** of the second instance.

    This one is an example of a "global behavior", because at minimum it
    affects two resource instance addresses and, if working with whole resource
    or whole module addresses, can potentially affect a large number of resource
    instances all at once.

### Provider-driven Behaviors

Providers get an opportunity to activate some special behaviors for a particular
resource instance when they respond to the `PlanResourceChange` function of
the provider plugin protocol.

When Terraform Core executes this RPC, it has already selected between
**Create**, **Delete**, or **Update** actions for the particular resource
instance, and so the special behaviors a provider may activate will typically
serve as modifiers or tweaks to that base action, and will not allow
the provider to select another base action altogether. The provider wire
protocol does not talk about the action types explicitly, and instead only
implies them via other content of the request and response, with Terraform Core
making the final decision about how to react to that information.

The following is a non-exhaustive list of existing examples of
provider-driven behaviors, selected to illustrate some different variations
that might be useful inspiration for new designs:

- When the base action is **Update**, a provider may optionally return one or
  more paths to attributes which have changes that the provider cannot
  implement as an in-place update due to limitations of the remote system.

    In that case, Terraform Core will replace the **Update** action with one of
    the two **Replace** variants, which means that from the provider's
    perspective the apply phase will really be two separate calls for the
    decomposed **Create** and **Delete** actions (in either order), rather
    than **Update** directly.
- When the base action is **Update**, a provider may optionally return a
  proposed new object where one or more of the arguments has its value set
  to what was in the prior state rather than what was set in the configuration.
  This represents any situation where a remote system supports multiple
  different serializations of the same value that are all equivalent, and
  so changing from one to another doesn't represent a real change in the
  remote system.

    If all of those taken together causes the new object to match the prior
    state, Terraform Core will treat the update as a **No-op** instead.

Of the three genres of special behaviors, provider-driven behaviors is the one
we've made the least use of historically but one that seems to have a lot of
opportunities for future exploration. Provider-driven behaviors can often be
ideal because their effects appear as if they are built in to Terraform so
that "it just works", with Terraform automatically deciding and explaining what
needs to happen and why, without any special effort on the user's part.

### Single-run Behaviors

Terraform Core's "plan" operation takes a set of arguments that we collectively
call "plan options", that can modify Terraform's planning behavior on a per-run
basis without any configuration changes or special provider behaviors.

As noted above, this particular genre of designs is the most burdensome to
implement because any wrapping software that can ask Terraform Core to create
a plan must ideally offer some way to set all of the available planning options,
or else some part of Terraform's functionality won't be available to anyone
using that wrapper.

However, we've seen various situations where single-run behaviors really are the
most appropriate way to handle a particular use-case, because the need for the
behavior originates in some process happening outside of the scope of any
particular Terraform module or provider.

The following is a non-exhaustive list of existing examples of
single-run behaviors, selected to illustrate some different variations
that might be useful inspiration for new designs:

- The "replace" planning option specifies zero or more resource instance
  addresses.

    For any resource instance specified, Terraform Core will transform any
    **Update** or **No-op** action for that instance into one of the
    **Replace** actions, thereby allowing an operator to respond to something
    having become degraded in a way that Terraform and providers cannot
    automatically detect and force Terraform to replace that object with
    a new one that will hopefully function correctly.
- The "refresh only" planning mode ("planning mode" is a single planning option
  that selects between a few mutually-exclusive behaviors) forces Terraform
  to treat every resource instance as **No-op**, regardless of what is bound
  to that address in state or present in the configuration.

## Legacy Operations

Some of the legacy operations Terraform CLI offers that _aren't_ integrated
with the plan and apply flow could be thought of as various degenerate kinds
of single-run behaviors. Most don't offer any opportunity to preview an effect
before applying it, but do meet a similar set of use-cases where an operator
needs to take some action to respond to changes to the context Terraform is
in rather than to the Terraform configuration itself.

Most of these legacy operations could therefore most readily be translated to
single-run behaviors, but before doing so it's worth researching whether people
are using them as a workaround for missing configuration-driven and/or
provider-driven behaviors. A particular legacy operation might be better
replaced with a different sort of special behavior, or potentially by multiple
different special behaviors of different genres if it's currently serving as
a workaround for many different unmet needs.
