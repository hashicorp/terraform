---
layout: "docs"
page_title: "Understanding the Schema"
sidebar_current: "docs-internals-provider-guide-schema"
description: |-
  Details about the schema that defines your resource or data source.
---

# Understanding the Schema

The `*schema.Schema` type is central to the way Terraform's providers work, so
it's worth exploring some of the advanced tools providers have at their
disposal to help Terraform do the right thing. The `*schema.Schema` can be
reductively explained as the "type system" of a Terraform resource or data
source, but it goes a little deeper than that. It has all the information
expected of a type description: the type of information (a string, an integer,
a list, etc.) that it expects, whether that information is optional or
computed, etc.; but it also has a bunch of special features that can be defined
for each field.

## Working with Computed

There are some fields on an resource or data source that can't be influenced by
the user at all, which the API provider is the source of truth for. Examples
include the timestamp a resource was created, or the ID of a resource. For
these special fields, Terraform wants to notice when users mistakenly set them
in config files and throw an error, instead of silently ignoring them. To make
this possible, any resource or data source property that should be stored in
state but that the user should not be able to configure should have its
`Computed` property set to `true`.

To complicate things, there are some fields on a resource that the user _can_
set, but if they don't, the API will pick a value for them. Examples include
the IP address assigned to a compute instance, or version of a disk image to
use when creating a disk. For these properties, Terraform wants to know that if
the config file doesn't ask for anything specific, the user is happy with
whatever the server returns, but if there is something in the config file,
Terraform needs to ensure the server reflects that value. To make this
possible, any resource property that the server gets to pick a default for but
the user should be able to override should have _both_ its `Computed` and
`Optional` properties set to `true`. If the server does not respect the value
the user asks for and generates one on its own, Terraform will consider that a
diff, and will keep trying to correct it. This often leads to perpetual diff
bugs, so it's important that only properties the user can actually _set_ have
their `Optional` property set to `true`.

## Working with `ForceNew`

Some resources are completely immutable - if anything about them changes,
Terraform needs to tear them down and build up again from scratch. Sometimes
properties need to be set on resources when they're created, and can't be
changed after. For example, a compute instance's region is set when it's
started, and once set, it can't be changed.

To help in this common scenario, resource fields have a `ForceNew` property
that, when set to `true`, indicates to Terraform that if it notices a diff, it
should just tear down the old one and stand up a new one.

## Deprecating and Removing Properties

Sometimes a Provider supports input that can't be supported later. Sometimes a
field needs to be renamed. In these situations, Terraform provides two
properties that can optionally be set on any field for a resource.

If `Deprecated` is set to anything but the empty string, when a config file
sets or references that field, Terraform will display that string to the user.
This is useful for notifying users that a field is going away or should no
longer be used without actually breaking their config. A good value for this
property would be a message indicating that the field is deprecated, and
pointing the user to suggestions for what to do instead and/or more
information.

If `Removed` is set to anything but the empty string, when a config file sets
or references that field, Terraform will throw an error, stop execution, and
display that string to the user. This is useful for offering users more context
or information, instead of a config field just disappearing on them. A good
value for this property would be a message indicating that the field has been
removed, and pointing the user to suggestions for what to do instead and/or
more information.

## Working with Defaults

There are times when it's desirable to allow a user to set the value of a
field, but if they don't, the empty value is not optimal. For example, a
resource could have a default description if a user doesn't override it, or a
default location for credentials. Terraform offers a few ways to handle these
situations, depending on what a provider needs.

The `Default` property of the `*schema.Schema` supports a hard-coded default
value. If the user does not provide a value for the field, Terraform will use
the value of the `Default` property, instead, and the user will not be prompted
for input.

A The `DefaultFunc` property of the `*schema.Schema` can be set with a
[`schema.SchemaDefaultFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaDefaultFunc),
which should return the value to be set or an error. It's important the output
of `DefaultFunc` be stable and return the same values if the config hasn't
changed; otherwise, users will see a diff every time the returned value
changes, even though they haven't changed anything.

Only one of `Default` or `DefaultFunc` may be set on any given field.

There is also an `InputDefault` property on `*schema.Schema`. Unlike `Default`
or `DefaultFunc`, the user will still be prompted for input if `InputDefault`
is set and they don't specify a value. The value of `InputDefault` will be
pre-populated in the input for the user.

Defaults can pose problems for backwards compatibility inside providers.
Because `Default` and the value provided by `DefaultFunc` are stored in state
and evaluated on every run, changing `Default` or the value provided by
`DefaultFunc` results in a diff for any user that has the previous value in
their state. Defaults can also pose problems when importing, as the default
will not be set on the imported resource, so the resource winds up in a state
that it could not have been put in by a config, and an apply is necessary
before it can be fixed. When a field that has a default has
[`ForceNew`](#working-with-forcenew) set to `true` on it, as well, this is
compounded. [State migrations](#versioning-the-schema) can mitigate this, as
can the use of [`Computed`](#working-with-computed), but carefully considered
defaults will pay off in terms of backwards compatibility.

## Working with Sets and Lists

Sometimes Providers needs to model fields that accept a group of values, not
just a single value. In these cases, either a set or a list is appropriate. A
rule of thumb is that if order is not important, a set is the right type.
Otherwise, use a list. If in doubt, use a list. One thing to know is that a set
can only have one of each value in it; if the same value could appear in the
group twice, use a list.

To make a field accept a set, set its `Type` property to `schema.TypeSet`. The
`Set` property of the field can be set to a
[`schema.SchemaSetFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc)
to customize the way values are hashed into keys, but unless it can be proven
necessary, leaving it empty and using the default hashing function is
recommended.

To make a field accept a list, set its `Type` property to `schema.TypeList`.

Regardless of whether the field accepts a list or a set, it also needs its
`Elem` property set to an instance of a `*schema.Schema` or a
`*schema.Resource`. A `*schema.Schema` will just be a simple value, but a
`*schema.Resource` houses a more complex structure, and could have its own
lifecycle for elements being added and removed.

When using a `*schema.Resource` as the `Elem` for a set, it's important to know
that fields on the `*schema.Resource` that have their `Computed` property set
to `true` will cause problems. This should be avoided. Nesting a set inside of
another set should also be avoided, as changing the contents of the inner set
will mark the outer set as changed, which leads to some hard-to-understand
diffs.

It's difficult to know the key for set items ahead of time, so it will be hard
to interpolate specific items of the set. Lists do not have this problem, as
the key is always their position in the list.

Inserting a new item into the list at any point but the end of the list will
cause every item following the newly inserted item to think it has "changed",
even though only its position in the list has changed. When the entire list can
be updated atomically, this isn't such a big deal, but if updating the list
involves making an API call for each element that has changed, this can get
messy.

It iss easier to switch from a list to a set than to switch from a set to a
list, as transitioning from a set to a list means trying to find a way to order
the items in the set. Transitioning from a list to a set just means discarding
the order.

The optional [validation function](#customizing-validation) for fields not work
with lists or sets.

## Working With Either/Or Fields

There are times when a field in a resource conflicts with one or more other
fields, and only one should be set at a time. For example, a health check may
accept an HTTP endpoint, an HTTPS endpoint, or a TCP endpoint, but only one of
the three.

To help in this scenario, the `*schema.Schema`'s `ConflictsWith` property can
be set. Its value should be a list of the fields that the property conflicts
with. It's important to note that _each_ field must have _all_ the fields it
conflicts with set for this to work appropriately. For example, if A conflicts
with B, A's `ConflictsWith` property should be set to `"B"`, and `B`'s
`ConflictsWith` property should be set to `"A"`.

If the user sets a field that the given field has listed in its `ConflictsWith`
property, Terraform will return an error and exit execution.

## Customizing State

Sometimes it's not optimal to store the state _exactly_ as it's represented in
`*schema.ResourceData`. For example, for large strings, it may be better to
only store the hash of the string.

For these cases, the `StateFunc` property of the `*schema.Schema` can be used
to customize the way a field appears in state. This function will also be
called, with the config file's values as input, before comparing the state for
diffing.

To take advantage of this behaviour, set the `StateFunc` property to a
[`schema.SchemaStateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaStateFunc).
The interface will be the value that would be written to state, and the
`schema.SchemaStateFunc` needs to return the string it should be represented as
in state.

## Customizing Validation

While Terraform will do its best to ensure users are entering the right type of
inputs for a schema, its capabilities are pretty limited unless providers give
it some information.

One of `Required`, `Optional`, or [`Computed`](#working-with-computed) must be
set to `true`. `Required` and `Optional` cannot both be set to `true`. If
`Required` is `true`, Terraform will automatically return an error if the field
is not set. If `Optional` is `true`, Terraform will not object to having an
empty value (or no value at all) set for the field. If only `Computed` is set
to `true`, Terraform will return an error if the user tries to enter any input
at all for that field; this is most useful for fields controlled entirely by
the API.

For fields requiring even further control, the `*schema.Schema` has a
`ValidateFunc` property. This can be set to a
[`schema.SchemaValidateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaValidateFunc),
which allows providers to define arbitrary validation logic and return either
errors (which will halt Terraform execution) or warnings (which will not halt
execution, but will be displayed in the output).

~> **Note:** `ValidateFunc` only works on primitive types; sets and lists
cannot use it for validation.

Validation is a bit of an art form. The stricter the validation applied, the
sooner users will be notified of errors, and the more errors will be caught
before any resources are created. But stricter validation also runs the risk of
the API changing its validation logic, and the provider calling valid input
invalid, frustrating users. A good rule of thumb is to apply validation when
the API's validation is fairly static and unlikely to change.

## Customizing Diffs

Some forms of data may _look_ different, but are semantically the same. In
these cases, Terraform will think that something has changed, when in reality,
it hasn't. A great example of this is JSON data, which can be represented any
number of ways, but still mean the same thing.

For these cases, it's a bad user experience to ignore semantic equality and let
Terraform just enforce lexicographic equality, especially if the API isn't
consistent about its output format, or if the field in question has
[`ForceNew`](#working-with-forcenew) set to `true`.

Instead, use the `DiffSuppressFunc` property of the `*schema.Schema` to supply
a
[`schema.SchemaDiffSuppressFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaDiffSuppressFunc)
to help Terraform understand which values should be considered equal. The
function will be called with the key of the field in question, the value in
state, the new value, and the `*schema.ResourceData` the new value came from.
It should return `true` if the values should be considered equal (and
therefore, not a diff), or false if the values should be considered unequal
(and therefore, a diff).

~> **Note:** `DiffSuppressFunc` should only be used on primitive types; sets
and lists will be challenging to use it correctly with.

## Storing Sensitive Information

Terraform's state may sometimes hold sensitive information. These fields should
not be displayed in all the diffs or to anyone that runs `terraform show`. To
prevent fields from being shown in output, set the `Sensitive` property of
their `Schema` to `true`.

~> **Note:** This _does not_ protect the information in state. In the future it
may, but for now, it's important to know how to [protect
statefiles](/docs/state/sensitive-data.html).

## Versioning the Schema

Maintaining backwards compatibility is an important part of maintaining a
Terraform provider. Because Terraform manages infrastructure, it can be
expensive and disruptive to have a breaking change.

To help with this, the state associated with each of the resources Terraform
manages is versioned, and providers can increment the version numbers and tell
Terraform how to migrate the state of existing users. This allows the shape of
the state to change over time while automating away the upgrade process for
users.

This generally is only needed when providers change the way an existing field
is persisted in state; adding new fields and removing fields almost always
works without a migration. It should be fairly straightforward to test if a
migraiton is necesary: set up state using the previous version of a provider,
then upgrade to the new version, and try using `terraform plan` or `terraform
apply`. If there are any unexpected diffs or errors, a state migration may be
necessary.

To use the migration functionality, increment the `SchemaVersion` property of
the `*schema.Resource`. If it's not set, set it to `1`.

The `MigrateState` property should be set to a
[`schema.StateMigrateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#StateMigrateFunc),
which defines the logic for migrating the state of the `*schema.Resource`.

The `schema.StateMigrateFunc` will be passed the version of the schema the
state is currently in, a pointer to the
[`terraform.InstanceState`](https://godoc.org/github.com/hashicorp/terraform/terraform#InstanceState)
that needs to be migrated to the new format, and the `interface{}` returned by
the provider's [`ConfigureFunc`](new-provider.html#instantiating-clients).

The version the state is in is important, as a migration function should allow
users of _any_ previous state (not just the most recent one) to migrate to the
new format.

The `terraform.InstanceState` can be inspected and set values (using its
`Attributes` property, which is a map with field names as keys and
`interface{}` stored in state for that field as values).

The `interface{}` returned by the provider's `ConfigureFunc` is useful in case
API calls are required to properly migrate the state.

The function should return the new `*terraform.InstanceState`, or an `error` if
the state could not be migrated.

This function will be run during the refresh portion of the Terraform
lifecycle, and so should get executed before any of the provider's lifecycle
functions are called.
