# Terraform Resource Instance Change Lifecycle

This document describes the relationships between the different operations
called on a Terraform Provider to handle a change to a resource instance.

![](https://gist.githubusercontent.com/apparentlymart/c4e401cdb724fa5b866850c78569b241/raw/fefa90ce625c240d5323ea28c92943c2917e36e3/resource_instance_change_lifecycle.png)

The process includes several different artifacts that are all objects
conforming to the schema of the resource type in question, representing
different subsets of the instance for different purposes:

* **Configuration**: Contains only values from the configuration, including
  unknown values in any case where the argument value is derived from an
  unknown result on another resource. Any attributes not set directly in the
  configuration are null.

* **Prior State**: The full object produced by a previous apply operation, or
  null if the instance is being created for the first time.

* **Proposed New State**: Terraform Core merges the non-null values from
  the configuration with any computed attribute results in the prior state
  to produce a combined object that includes both, to avoid each provider
  having to re-implement that merging logic. Will be null when planning a
  delete operation.

* **Planned New State**: An approximation of the result the provider expects
  to produce when applying the requested change. This is usually derived from
  the proposed new state by inserting default attribute values in place of
  null values and overriding any computed attribute values that are expected
  to change as a result of the apply operation. May include unknown values
  for attributes whose results cannot be predicted until apply. Will be null
  when planning a delete operation.

* **New State**: The actual result of applying the change, with any unknown
  values from the planned new state replaced with final result values. This
  value will be used as the input to plan the next operation.

The remaining sections describe the three provider API functions that are
called to plan and apply a change, including the expectations Terraform Core
enforces for each.

For historical reasons, the original Terraform SDK is exempt from error
messages produced when the assumptions are violated, but violating them will
often cause downstream errors nonetheless, because Terraform's workflow
depends on these contracts being met.

The following section uses the word "attribute" to refer to the named
attributes described in the resource type schema. A schema may also include
nested blocks, which contain their _own_ set of attributes; the constraints
apply recursively to these nested attributes too.

Nested blocks are a configuration-only construct and so the number of blocks
cannot be changed on the fly during planning or during apply: each block
represented in the configuration must have a corresponding nested object in
the planned new state and new state, or an error will be returned.

If a provider wishes to report about new instances of the sub-object type
represented by nested blocks that are created implicitly during the apply
operation -- for example, if a compute instance gets a default network
interface created when none are explicitly specified -- this must be done via
separate `Computed` attributes alongside the nested blocks, which could for
example be a list or map of objects that includes a mixture of the objects
described by the nested blocks in the configuration and any additional objects
created by the remote system.

## ValidateResourceTypeConfig

`ValidateResourceTypeConfig` is the provider's opportunity to perform any
custom validation of the configuration that cannot be represented in the schema
alone.

In principle the provider can require any constraint it sees fit here, though
in practice it should avoid reporting errors when values are unknown (so that
the operation can proceed and determine those values downstream) and if
it intends to apply default values during `PlanResourceChange` then it must
tolerate those attributes being null at validation time, because validation
happens before planning.

A provider should repeat similar validation logic at the start of
`PlanResourceChange`, in order to catch any new
values that have switched from unknown to known along the way during the
overall plan/apply flow.

## PlanResourceChange

The purpose of `PlanResourceChange` is to predict the approximate effect of
a subsequent apply operation, allowing Terraform to render the plan for the
user and to propagate any predictable results downstream through expressions
in the configuration.

The _planned new state_ returned from the provider must meet the following
constraints:

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

`PlanResourceChange` is actually called twice for each resource type.
It will be called first during the planning phase before Terraform prints out
the diff to the user for confirmation. If the user accepts the plan, then
`PlanResourceChange` will be called _again_ during the apply phase with any
unknown values from configuration filled in with their final results from
upstream resources. The second planned new state is compared with the first
and must meet the following additional constraints along with those listed
above:

* Any attribute that had a known value in the first planned new state must
  have an identical value in the second.

* Any attribute that had an unknown value in the first planned new state may
  either remain unknown in the second or take on any known value of the
  expected type.

It is the second planned new state that is finally provided to
`ApplyResourceChange`, as described in the following section.

## ApplyResourceChange

The `ApplyResourceChange` function is responsible for making calls into the
remote system to make remote objects match the planned new state. During that
operation, it should determine final values for any attributes that were left
unknown in the planned new state, thus producing a wholly-known _new state_
object.

`ApplyResourceChange` also recieves the prior state so that it can use it
to potentially implement more "surgical" changes to particular parts of
the remote objects by detecting portions that are unchanged, in cases where the
remote API supports partial-update operations.

The new state object returned from the provider must meet the following
constraints:

* Any attribute that had a known value in the planned new state must have an
  identical value in the new state.

* Any attribute that had an unknown value in the planned new state must take
  on a known value of the expected type in the new state. No unknown values
  are allowed in the new state.
