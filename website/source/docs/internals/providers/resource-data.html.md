---
layout: "docs"
page_title: "Using ResourceData"
sidebar_current: "docs-internals-provider-guide-resource-data"
description: |-
  Details about the ubiquitous ResourceData type and all its options.
---

# Using `ResourceData`

When working with the `Create`, `Read`, `Update`, `Destroy`, and `Exists`
methods on your Provider, it's almost impossible to not run into
`ResourceData`. It's a type that is used all over, so it's important to
understand it. But `ResourceData` can also be tricky, because it's not
abstracting a single logical concept in Terraform.

## How to Conceptualise `ResourceData`

The most useful way to think of `ResourceData` is not as a placeholder for any
one concept in Terraform—it's not just information about the config or the
state, for example—but as the desired state of an object. If you think of
Terraform as the convergence of your config file and your infrastructure,
`ResourceData` is how you express  that understanding of what your
infrastructure should look like.

## What `ResourceData` Abstracts

To get practical, `ResourceData` abstracts four separate concepts into a single
source of truth:

* The current state
* The config file
* The diff being applied
* Any calls to `ResourceData.Set`

Those are expressed in the order of their priority. That is, the config
overrides any values set in the state, and every time you call `Set`, it
overrides everything else.

`ResourceData` often gets confused for either the state or the config, but it's
important to realise that it's a step removed from these concepts, and is used
more as an understanding of what the infrastructure _should_ be.

## Retrieving Properties

To retrieve properties from `ResourceData`, use the `Get` or `GetOk` methods.
`Get` takes a property name or address (e.g., `myprop.0.value`) and returns it
as an `interface{}`. It's important to note that the framework will always
guarantee consistency about the underlying type of that `interface{}`. If the
key doesn't exist in your schema, `Get` returns `nil`. If the key exists in
your schema, but not in the config, `Get` returns the empty value for that
type.

`ResourceData` also has a `GetOk` method that functions identically to its
`Get` method, but with an extra return parameter. The new return parameter
returns `true` if the property is set to a non-zero value, but with caveats.
This is where it's important to remember that `ResourceData` is an amalgamation
of input sources; providers cannot determine whether the property is set in the
config, in the state, in the diff, or by using `ResourceData.Set`. The only
information providers have available to them is that the property has been set
at some point, and what its value is right now.

## Detecting Changes

When there's a difference between what our infrastructure is and what we want
it to be, we want to be able to see what changed and what it should be. To aid
in this, `ResourceData` provides a `HasChange` method and a `GetChange` method.
Each takes a property key, just like `Get`. `HasChange` returns `true` if
there's a change to that property; the change could be from drift (someone or
something modifying the infrastructure outside of Terraform) or from config
changes. The `GetChange` method returns two values; a basic understanding is
that the first value is what the property _used to be_, and the second value is
what the property _should be changed to_.  A slightly more nuanced
understanding is that the first value is what is in the state (representing the
state of the infrastructure as it exists) and the second value is what is in
the config (representing what the user wants the infrastructure to be).

You'll notice there's no way to tell whether the state changed or the config
changed; this is a common misconception. Terraform never diffs states or
configs, it only ever diffs what is and what should be.

Be sure to call `HasChange` before calling `GetChange`; there are some cases
where `GetChange` would return both values as equal, but reflect a change. This
can happen, for example, when a boolean property has a default value of `false`
and was not specified, then gets set explicitly to `false`.

`GetChange` is only really necessary when you need to know what the previous
value of the field was; if you only need the new value, `Get` is sufficient.
`GetChange` is commonly used in cases where the API requires you to explicitly
remove and add items—for example, when there's a change in tags and the API
only offers `AddTag` & `RemoveTag` methods without a way to just change all
tags at once. In that case, you need to know what the previous tag value was,
so it can be removed.

## Setting State

At the end of your `Create`, `Get`, `Update`, and `Delete` functions, whatever
is in your `ResourceData` object is set as state. To manipulate this, the `Set`
method is provided. It takes the key of a property, just like `Get`, as an
argument, and the value to set it to.

It's important that all your properties get set using the `Set` method, as
otherwise Terraform will be unable to perform some of its important functions,
but will give the appearance of operating normally. For example, changes to the
config file will be detected, but changes made outside Terraform will be
silently ignored. This breaks Terraform's promise of reflecting your
infrastructure as code, so it's important that the current state of the
infrastructure get persisted using the `Set` method.

When using the `Set` method, you shouldn't dereference pointer values, as doing
so incorrectly (for example, when they're set to a nil value) can crash
Terraform. The `Set` method is capable of safely derefencing on its own, so the
safest course of action is to just pass it the pointer.

## Partial Updates

Some resources and data sources cannot be managed using only a single API call;
for example, a resource may need its permissions set or a compute instance may
need a disk attached to it, even though the instance and disk are modeled as a
single resource.

In most cases, this works fine. Even in the case of failures, a the refresh
part of the lifecycle fixes this. But in some cases (like when refresh is
disabled), it's helpful to be able to gracefully recover from a failure. For
these cases, the `*ResourceData` type has
[`SetPartial`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ResourceData.SetPartial)
and
[`Partial`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ResourceData.Partial)
methods. Before we get into how they change Terraform's behaviour, let's talk
about how Terraform usually works, again.

Usually, when a resource is created or updated, Terraform
[builds](#what-resourcedata-abstracts) the `*ResourceData` from the current
state, the config file, and the diff being applied. It then calls your `Create`
or `Update` function (as appropriate) and any calls to the `*ResourceData`'s
`Set` method get layered on top of the `*ResourceData`. After _successful_
completion of the function, the `*ResourceData` gets stored as the new state.
Should the function return an error, the `*ResourceData` gets discarded and is
_not_ set as the state.

The `Partial` method accepts a boolean designating whether partial state mode
is on or off. By default, it's off. When it's off, the persisting of
`*ResourceData` to state works as normal, and `SetPartial` has no effect. When
partial state mode is on, however, _only_ the fields persisted with
`SetPartial` are persisted to state, and they're persisted whether the function
errors or not. So you could use `Partial` to turn on partial state mode, make
one API request, and if it's successful, call `SetPartial` to set its values
before making the second API request. If the second API request fails,
Terraform will still have the results of the first request in state, and will
not attempt to redo that API call. When all your API calls have completed
successfully, you should turn partial state mode off, so the normal state
persistence rules will apply and everything that came from the config, diff,
and previous state will still be persisted.

Note that `SetPartial` can only operate on root keys. You cannot use it to set
individual items in a list or a set.

## Working With IDs

Terraform uses IDs to reference resources. It's part of the key used when
accessing the resource, either in the provider or in interpolation. It's also
what the user supplies when running [`terraform
import`](/docs/import/index.html) to identify the resource they wish to import.
A good ID is immutable and easy for a user to locate.

Terraform special-cases the `id` property on resources. Do not use `id` as a
key for a property. Terraform uses that internally, and trying to access an
`id` property with the `Get` method will always return a zero value.

To retrieve the ID, use the `GetId` method on `ResourceData`. To set it, use
the `SetId` method, passing the string to use as an ID. Only strings may be
used as IDs.

You should always set the ID as soon as you possibly can. Even with
asynchronous APIs, you shouldn't wait for the API to finish creating the
resource, just set the ID as soon as it's provided by the API. This is
important because Terraform needs to record the fact that the resource exists,
even if other Terraform calls fail and cause Terraform to exit. Otherwise, the
ID could remain empty and leave the user in a situation where the resource was
created, but Terraform doesn't know about it.
