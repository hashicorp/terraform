---
layout: "docs"
page_title: "Using ResourceData"
sidebar_current: "docs-internals-provider-guide-resource-data"
description: |-
  Details about the ubiquitous ResourceData type and all its options.
---

# Using `ResourceData`

The `schema.ResourceData` is used throughout the provider lifecycle, and is the
main way to interact with both config files and state. This makes it an
important type to understand, but the abstraction also makes it difficult to
conceptualize.

## How to Conceptualize `ResourceData`

The `schema.ResourceData` type does not map directly onto any single concept in
Terraform. It's not, for example, a representation of the config file, or of
the state. It's a representation of the desired state of a resource. The
`schema.ResourceData` is how Terraform expresses its understanding of what the
infrastructure should look like once Terraform has done its job.

## What `ResourceData` Abstracts

To get practical, `ResourceData` abstracts four separate concepts into a single
source of truth:

* The current state
* The config file
* The diff being applied
* Any calls to `ResourceData.Set`

Those are expressed in the order of their priority. The config overrides any
values set in the state, and `Set` invocations override everything else.

`ResourceData` often gets confused for either the state or the config, but it's
important to realise that it's a step removed from these concepts, and is used
more as an understanding of what the infrastructure _should_ be.

## Retrieving Fields

To retrieve fields from `schema.ResourceData`, use the `Get` or `GetOk`
methods. `Get` takes a field name or address (e.g., `myprop.0.value`) and
returns it as an `interface{}`. It's important to note that the framework will
always guarantee consistency about the underlying type of that `interface{}`.
If the key doesn't exist in your schema, `Get` returns `nil`. If the key exists
in the schema, but not in the config, `Get` returns the empty value for that
type.

`schema.ResourceData` also has a `GetOk` method that functions similarly to the
`Get` method, but with an extra return parameter. The extra return parameter
returns `true` if the field is set to a non-zero value, but with caveats. It's
important to remember that `ResourceData` is an amalgamation of input sources;
providers cannot determine whether the field is set in the config, in the
state, in the diff, or by using `ResourceData.Set`. The only information
providers have available to them is that the field has been set at some point,
and what its value is right now.

## Detecting Changes

When the infrastructure is not as it should be, the `schema.ResourceData` uses
its `HasChange` and `GetChange` methods to know what corrections need to be
made. Each takes a field key, just like `Get`. `HasChange` returns true if
there's a change to that field; the change could be from drift (someone or
something modifying the infrastructure outside of Terraform) or from config
changes. The `GetChange` method returns the value of the field in the state and
the value of the field in the config.

There's no way to tell whether the state or the config changed. Terraform does
not compare two different versions of state, or two different versions of the
config. It only tracks the most recent version of state and config, and reports
whether there's a difference between those two things or not.

`HasChange` must be called before `GetChange`, as `GetChange` may return two
equal values but still represent a change. For example, a boolean field with a
default value of `false` that was not specified that gets explicitly set to
false would return `true` from `HasChange` but two equal values from
`GetChange`.

`GetChange` is only necessary when the value in state is needed. If only the
desired value is needed, `Get` is sufficient. `GetChange` is commonly used in
cases where items in a set or list must be explicitly added or removed, with no
way to set all the items at once. In that case, it's necessary to differentiate
between the config and the state, to know what needs to be explicitly removed
and what needs to be added.

## Setting State

Whatever is in the `*schema.ResourceData` at the end of the `Create`, `Get`,
`Update`, and `Delete` functions is set as the new state of that resource. The
`Set` method updates the value of a key in the `*schema.ResourceData`. The
method takes the key of a field, just like `Get`, and the value to set it to.

All possible fields in a resource should be specified using the `Set` method,
as otheriwse Terraform cannot preform some of its important functions. For
example, changes to the config file will be detected, but changes made outside
Terraform will be silently ignored. This breaks Terraform's promise of
representing infrastructure as code, so it's important that the current state
of the infrastructure get persisted using the `Set` method.

When using the `Set` method, pointer values should not be dereferenced. Doing
so incorrectly (for example, when they're set to a nil value) can crash
Terraform, and the `Set` method is capable of safely derefencing on its own.
The safest course of action is to just pass the pointers to `Set` and allow it
to dereference if necessary.

## Partial Updates

Some resources and data sources cannot be managed using only a single API call.
A resource may need its permissions set or a compute instance may need a disk
attached, even though the instance and disk are modeled as a single resource.

In most cases, this works fine. Even in the case of failures, the refresh part
of the lifecycle fixes this. But in some cases (like when refresh is disabled),
it's helpful to be able to gracefully recover from a failure. For these cases,
the `*schema.ResourceData` type has
[`SetPartial`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ResourceData.SetPartial)
and
[`Partial`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ResourceData.Partial)
methods.

Normally, when a resource is created or updated, Terraform
[builds](#what-resourcedata-abstracts) the `*schema.ResourceData` from the
current state, the config file, and the diff being applied. It then calls the
`schema.CreateFunc` or `schema.UpdateFunc` function (as appropriate) and any
calls to the `*schema.ResourceData`'s `Set` method get layered on top of the
`*schema.ResourceData`. After _successful_ completion of the function, the
`*ResourceData` gets stored as the new state. Should the function return an
error, the `*ResourceData` gets discarded and is _not_ set as the state.

The `Partial` method accepts a boolean designating whether partial state mode
is on or off. By default, it's off. When it's off, the persisting of
`*schema.ResourceData` to state works as normal, and `SetPartial` has no
effect. When partial state mode is on, however, _only_ the fields persisted
with `SetPartial` are persisted to state, and they're persisted whether the
function errors or not. Functions generally use `Partial` to turn on partial
state mode, make one API request, and if it's successful, call `SetPartial` to
set its values before making the second API request. If the second API request
fails, Terraform will still have the results of the first request in state, and
will not attempt to redo that API call. When all the API calls have completed
successfully, the function should turn partial state mode off, so the normal
state persistence rules will apply and everything that came from the config,
diff, and previous state will still be persisted.

Note that `SetPartial` can only operate on root keys. It cannot set individual
items in a list or a set.

## Working With IDs

Terraform uses IDs to reference resources. It's part of the key used when
accessing the resource, either in the provider or in interpolation. It's also
what the user supplies when running [`terraform
import`](/docs/import/index.html) to identify the resource they wish to import.
A good ID is immutable and easy for a user to locate.

Terraform special-cases the `id` field on resources. It cannot be used as the
key for a field, as trying to access an `id` field with the `Get` method will
always return a zero value.

To retrieve the ID, use the `GetId` method on `*schema.ResourceData`. To set
it, use the `SetId` method, passing the string to use as an ID. Only strings
may be used as IDs.

IDs should be persisted as soon possible. Even with asynchronous APIs, don't
wait for the API to finish creating the resource before setting the ID.
Terraform needs to record that the resource exists, even if other Terraform
calls fail and cause Terraform to exit. Otherwise, the ID could remain empty
and leave the user in a situation where the resource was created, but Terraform
doesn't know about it.
