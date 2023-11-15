# Terraform Resource Instance Change Lifecycle

This document describes the relationships between the different operations
called on a Terraform Provider to handle a change to a resource instance.

![](https://user-images.githubusercontent.com/20180/172506401-777597dc-3e6e-411d-9580-b192fd34adba.png)

The resource instance operations all both consume and produce objects that
conform to the schema of the selected resource type.

The overall goal of this process is to take a **Configuration** and a
**Previous Run State**, merge them together using resource-type-specific
planning logic to produce a **Planned State**, and then change the remote
system to match that planned state before finally producing the **New State**
that will be saved in order to become the **Previous Run State** for the next
operation.

The various object values used in different parts of this process are:

* **Configuration**: Represents the values the user wrote in the configuration,
  after any automatic type conversions to match the resource type schema.

    Any attributes not defined by the user appear as null in the configuration
    object. If an argument value is derived from an unknown result of another
    resource instance, its value in the configuration object could also be
    unknown.

* **Prior State**: The provider's representation of the current state of the
  remote object at the time of the most recent read.

* **Proposed New State**: Terraform Core uses some built-in logic to perform
  an initial basic merger of the **Configuration** and the **Prior State**
  which a provider may use as a starting point for its planning operation.

    The built-in logic primarily deals with the expected behavior for attributes
    marked in the schema as "computed". If an attribute is only "computed",
    Terraform expects the value to only be chosen by the provider and it will
    preserve any Prior State. If an attribute is marked as "computed" and
    "optional", this means that the user may either set it or may leave it
    unset to allow the provider to choose a value.

    Terraform Core therefore constructs the proposed new state by taking the
    attribute value from Configuration if it is non-null, and then using the
    Prior State as a fallback otherwise, thereby helping a provider to
    preserve its previously-chosen value for the attribute where appropriate.

* **Initial Planned State** and **Final Planned State** are both descriptions
  of what the associated remote object ought to look like after completing
  the planned action.

    There will often be parts of the object that the provider isn't yet able to
    predict, either because they will be decided by the remote system during
    the apply step or because they are derived from configuration values from
    other resource instances that are themselves not yet known. The provider
    must mark these by including unknown values in the state objects.

    The distinction between the _Initial_ and _Final_ planned states is that
    the initial one is created during Terraform Core's planning phase based
    on a possibly-incomplete configuration, whereas the final one is created
    during the apply step once all of the dependencies have already been
    updated and so the configuration should then be wholly known.

* **New State** is a representation of the result of whatever modifications
  were made to the remote system by the provider during the apply step.

    The new state must always be wholly known, because it represents the
    actual state of the system, rather than a hypothetical future state.

* **Previous Run State** is the same object as the **New State** from
  the previous run of Terraform. This is exactly what the provider most
  recently returned, and so it will not take into account any changes that
  may have been made outside of Terraform in the meantime, and it may conform
  to an earlier version of the resource type schema and therefore be
  incompatible with the _current_ schema.

* **Upgraded State** is derived from **Previous Run State** by using some
  provider-specified logic to upgrade the existing data to the latest schema.
  However, it still represents the remote system as it was at the end of the
  last run, and so still doesn't take into account any changes that may have
  been made outside of Terraform.

* The **Import ID** and **Import Stub State** are both details of the special
  process of importing pre-existing objects into a Terraform state, and so
  we'll wait to discuss those in a later section on importing.


## Provider Protocol API Functions

The following sections describe the three provider API functions that are
called to plan and apply a change, including the expectations Terraform Core
enforces for each.

For historical reasons, the original Terraform SDK is exempt from error
messages produced when certain assumptions are violated, but violating them
will often cause downstream errors nonetheless, because Terraform's workflow
depends on these contracts being met.

The following section uses the word "attribute" to refer to the named
attributes described in the resource type schema. A schema may also include
nested blocks, which contain their _own_ set of attributes; the constraints
apply recursively to these nested attributes too.

The following are the function names used in provider protocol version 6.
Protocol version 5 has the same set of operations but uses some
marginally-different names for them, because we used protocol version 6 as an
opportunity to tidy up some names that had been awkward before.

### ValidateResourceConfig

`ValidateResourceConfig` takes the **Configuration** object alone, and
may return error or warning diagnostics in response to its attribute values.

`ValidateResourceConfig` is the provider's opportunity to apply custom
validation rules to the schema, allowing for constraints that could not be
expressed via schema alone.

In principle a provider can make any rule it wants here, although in practice
providers should typically avoid reporting errors for values that are unknown.
Terraform Core will call this function multiple times at different phases
of evaluation, and guarantees to _eventually_ call with a wholly-known
configuration so that the provider will have an opportunity to belatedly catch
problems related to values that are initially unknown during planning.

If a provider intends to choose a default value for a particular
optional+computed attribute when left as null in the configuration, the
provider _must_ tolerate that attribute being unknown in the configuration in
order to get an opportunity to choose the default value during the later
plan or apply phase.

The validation step does not produce a new object itself and so it cannot
modify the user's supplied configuration.

### PlanResourceChange

The purpose of `PlanResourceChange` is to predict the approximate effect of
a subsequent apply operation, allowing Terraform to render the plan for the
user and to propagate the predictable subset of results downstream through
expressions in the configuration.

This operation can base its decision on any combination of **Configuration**,
**Prior State**, and **Proposed New State**, as long as its result fits the
following constraints:

* Any attribute that was non-null in the configuration must either preserve
  the exact configuration value or return the corresponding attribute value
  from the prior state. (Do the latter if you determine that the change is not
  functionally significant, such as if the value is a JSON string that has
  changed only in the positioning of whitespace.)

* Any attribute that is marked as computed in the schema _and_ is null in the
  configuration may be set by the provider to any arbitrary value of the
  expected type.

* If a computed attribute has any _known_ value in the planned new state, the
  provider will be required to ensure that it is unchanged in the new state
  returned by `ApplyResourceChange`, or return an error explaining why it
  changed. Set an attribute to an unknown value to indicate that its final
  result will be determined during `ApplyResourceChange`.

`PlanResourceChange` is actually called twice per run for each resource type.

The first call is during the planning phase, before Terraform prints out a
diff to the user for confirmation. Because no changes at all have been applied
at that point, the given **Configuration** may contain unknown values as
placeholders for the results of expressions that derive from unknown values
of other resource instances. The result of this initial call is the
**Initial Planned State**.

If the user accepts the plan, Terraform will call `PlanResourceChange` a
second time during the apply step, and that call is guaranteed to have a
wholly-known **Configuration** with any values from upstream dependencies
taken into account already. The result of this second call is the
**Final Planned State**.

Terraform Core compares the final with the initial planned state, enforcing
the following additional constraints along with those listed above:

* Any attribute that had a known value in the **Initial Planned State** must
  have an identical value in the **Final Planned State**.

* Any attribute that had an unknown value in the **Initial Planned State** may
  either remain unknown in the second _or_ take on any known value that
  conforms to the unknown value's type constraint.

The **Final Planned State** is what passes to `ApplyResourceChange`, as
described in the following section.

### ApplyResourceChange

The `ApplyResourceChange` function is responsible for making calls into the
remote system to make remote objects match the **Final Planned State**. During
that operation, the provider should decide on final values for any attributes
that were left unknown in the **Final Planned State**, and thus produce the
**New State** object.

`ApplyResourceChange` also receives the **Prior State** so that it can use it
to potentially implement more "surgical" changes to particular parts of
the remote objects by detecting portions that are unchanged, in cases where the
remote API supports partial-update operations.

The **New State** object returned from the provider must meet the following
constraints:

* Any attribute that had a known value in the **Final Planned State** must have
  an identical value in the new state. In particular, if the remote API
  returned a different serialization of the same value then the provider must
  preserve the form the user wrote in the configuration, and _must not_ return
  the normalized form produced by the provider.

* Any attribute that had an unknown value in the **Final Planned State** must
  take on a known value whose type conforms to the type constraint of the
  unknown value. No unknown values are permitted in the **New State**.

After calling `ApplyResourceChange` for each resource instance in the plan,
and dealing with any other bookkeeping to return the results to the user,
a single Terraform run is complete. Terraform Core saves the **New State**
in a state snapshot for the entire configuration, so it'll be preserved for
use on the next run.

When the user subsequently runs Terraform again, the **New State** becomes
the **Previous Run State** verbatim, and passes into `UpgradeResourceState`.

### UpgradeResourceState

Because the state values for a particular resource instance persist in a
saved state snapshot from one run to the next, Terraform Core must deal with
the possibility that the user has upgraded to a newer version of the provider
since the last run, and that the new provider version has an incompatible
schema for the relevant resource type.

Terraform Core therefore begins by calling `UpgradeResourceState` and passing
the **Previous Run State** in a _raw_ form, which in current protocol versions
is the raw JSON data structure as was stored in the state snapshot. Terraform
Core doesn't have access to the previous schema versions for a provider's
resource types, so the provider itself must handle the data decoding in this
upgrade function.

The provider can then use whatever logic is appropriate to update the shape
of the data to conform to the current schema for the resource type. Although
Terraform Core has no way to enforce it, a provider should only change the
shape of the data structure and should _not_ change the meaning of the data.
In particular, it should not try to update the state data to capture any
changes made to the corresponding remote object outside of Terraform.

This function then returns the **Upgraded State**, which captures the same
information as the **Previous Run State** but does so in a way that conforms
to the current version of the resource type schema, which therefore allows
Terraform Core to interact with the data fully for subsequent steps.

### ReadResource

Although Terraform typically expects to have exclusive control over any remote
object that is bound to a resource instance, in practice users may make changes
to those objects outside of Terraform, causing Terraform's records of the
object to become stale.

The `ReadResource` function asks the provider to make a best effort to detect
any such external changes and describe them so that Terraform Core can use
an up-to-date **Prior State** as the input to the next `PlanResourceChange`
call.

This is always a best effort operation because there are various reasons why
a provider might not be able to detect certain changes. For example:
* Some remote objects have write-only attributes, which means that there is
  no way to determine what value is currently stored in the remote system.
* There may be new features of the underlying API which the current provider
  version doesn't know how to ask about.

Terraform Core expects a provider to carefully distinguish between the
following two situations for each attribute:
* **Normalization**: the remote API has returned some data in a different form
  than was recorded in the **Previous Run State**, but the meaning is unchanged.

    In this case, the provider should return the exact value from the
    **Previous Run State**, thereby preserving the value as it was written by
    the user in the configuration and thus avoiding unwanted cascading changes to
    elsewhere in the configuration.
* **Drift**: the remote API returned data that is materially different from
  what was recorded in the **Previous Run State**, meaning that the remote
  system's behavior no longer matches what the configuration previously
  requested.

    In this case, the provider should return the value from the remote system,
    thereby discarding the value from the **Previous Run State**. When a
    provider does this, Terraform _may_ report it to the user as a change
    made outside of Terraform, if Terraform Core determined that the detected
    change was a possible cause of another planned action for a downstream
    resource instance.

This operation returns the **Prior State** to use for the next call to
`PlanResourceChange`, thus completing the circle and beginning this process
over again.

## Handling of Nested Blocks in Configuration

Nested blocks are a configuration-only construct and so the number of blocks
cannot be changed on the fly during planning or during apply: each block
represented in the configuration must have a corresponding nested object in
the planned new state and new state, or Terraform Core will raise an error.

If a provider wishes to report about new instances of the sub-object type
represented by nested blocks that are created implicitly during the apply
operation -- for example, if a compute instance gets a default network
interface created when none are explicitly specified -- this must be done via
separate "computed" attributes alongside the nested blocks. This could be list
or map of objects that includes a mixture of the objects described by the
nested blocks in the configuration and any additional objects created implicitly
by the remote system.

Provider protocol version 6 introduced the new idea of structural-typed
attributes, which are a hybrid of attribute-style syntax but nested-block-style
interpretation. For providers that use structural-typed attributes, they must
follow the same rules as for a nested block type of the same nesting mode.

## Import Behavior

The main resource instance change lifecycle is concerned with objects whose
entire lifecycle is driven through Terraform, including the initial creation
of the object.

As an aid to those who are adopting Terraform as a replacement for existing
processes or software, Terraform also supports adopting pre-existing objects
to bring them under Terraform's management without needing to recreate them
first.

When using this facility, the user provides the address of the resource
instance they wish to bind the existing object to, and a string representation
of the identifier of the existing object to be imported in a syntax defined
by the provider on a per-resource-type basis, which we'll call the
**Import ID**.

The import process trades the user's **Import ID** for a special
**Import Stub State**, which behaves as a placeholder for the
**Previous Run State** pretending as if a previous Terraform run is what had
created the object.

### ImportResourceState

The `ImportResourceState` operation takes the user's given **Import ID** and
uses it to verify that the given object exists and, if so, to retrieve enough
data about it to produce the **Import Stub State**.

Terraform Core will always pass the returned **Import Stub State** to the
normal `ReadResource` operation after `ImportResourceState` returns it, so
in practice the provider may populate only the minimal subset of attributes
that `ReadResource` will need to do its work, letting the normal function
deal with populating the rest of the data to match what is currently set in
the remote system.

For the same reasons that `ReadResource` is only a _best effort_ at detecting
changes outside of Terraform, a provider may not be able to fully support
importing for all resource types. In that case, the provider developer must
choose between the following options:

* Perform only a partial import: the provider may choose to leave certain
  attributes set to `null` in the **Prior State** after both
  `ImportResourceState` and the subsequent `ReadResource` have completed.

    In this case, the user can provide the missing value in the configuration
    and thus cause the next `PlanResourceChange` to plan to update that value
    to match the configuration. The provider's `PlanResourceChange` function
    must be ready to deal with the attribute being `null` in the
    **Prior State** and handle that appropriately.
* Return an error explaining why importing isn't possible.

    This is a last resort because of course it will then leave the user unable
    to bring the existing object under Terraform's management. However, if a
    particular object's design doesn't suit importing then it can be a better
    user experience to be clear and honest that the user must replace the object
    as part of adopting Terraform, rather than to perform an import that will
    leave the object in a situation where Terraform cannot meaningfully manage
    it.
