---
layout: "docs"
page_title: "Understanding the Schema"
sidebar_current: "docs-internals-provider-guide-schema"
description: |-
  Details about the schema that defines your resource or data source.
---

# Understanding the Schema

The concept of a Schema is pretty central to the way Terraform’s Providers
work, so it’s worth exploring some of the advanced tools Providers have at
their disposal to help Terraform do the right thing. The idea of the Schema,
itself, can be reductively explained as the “type system” of a Terraform
Resource or Data Source, but it goes a little deeper than that. It has all the
information you’d expect: the type of information (a string, an integer, a
list, etc.) that it expects, whether that information is optional or computed,
etc.; but it also has a bunch of special features that can be defined for each
property.

## Working With Computed

There are some fields on an Resource or Data Source that can’t be influenced by
the user at all, which the API provider is the source of truth for. Things like
the timestamp a resource was created, or the ID of a resource. For these
special properties, Terraform wants to notice when users mistakenly set them in
config files and throw an error, instead of silently ignoring them. To make
this possible, any Resource or Data Source property that should be stored in
state but that the user should not be able to configure should have its
`Computed` property set to true.

To complicate things, there are some fields on a Resource that the user _can_
set, but if they don’t, the API will pick a value for them. Things like the IP
address assigned to a compute instance, or version of a disk image to use when
creating a disk. For these properties, Terraform wants to know that if the
config file doesn’t ask for anything specific, the user is happy with whatever
the server returns, but if there is something in the config file, Terraform
needs to ensure the server reflects that value. To make this possible, any
Resource property that the server gets to pick a default for but the user
should be able to override should have _both_ its `Computed` and `Optional`
properties set to `true`. It’s important to note that if the server does not
respect the value the user asks for and generates one on its own, Terraform
will consider that a diff, and will keep trying to correct it. This often leads
to perpetual diff bugs, so it’s important that only properties the user can
actually _set_ have their `Optional` property set to `true`.

## Working With `ForceNew`

Some Resources are completely immutable--if you want to change anything about
them, you need to just tear them down and build up again from scratch.
Sometimes properties need to be set on Resources when they’re created, and
can’t be changed after. For example, you need to decide which region you want a
compute instance in before you stand it up, and once you create it, you can’t
change it&mdash;you can only tear down that instance and stand up a new one.

To help in this common scenario, Resources properties have a `ForceNew`
property that, when set to `true`,  indicates to Terraform that if it notices a
diff, it should just tear down the old one and stand up a new one.

## Deprecating and Removing Properties

Things change sometimes, and that’s okay. Sometimes a Provider supports input
that can’t be supported later. Sometimes a property needs to be renamed. In
these situations, Terraform provides two properties that can optionally be set
on any field for a Resource.

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

## Working With Defaults

There are times when it's desirable to allow a user to set the value of a
field, but if they don't, the empty value is not optimal. For example, you
could provide a default description for a resource if a user doesn't override
it, or a default location for credentials. Terraform offers a few ways to
handle these situations, depending on your needs.

The `Default` property of your `Schema` allows you to hardcode a default value.
If the user does not provide a value for the field, Terraform will use the
value you supply, instead, and the user will not be prompted for input.

The `DefaultFunc` property of your `Schema` allows you to specify a
[`schema.SchemaDefaultFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaDefaultFunc),
which should return the value to be set or an error. It's important the output
of `DefaultFunc` be stable and return the same values if the config hasn't
changed; otherwise, your users will see a diff every time the returned value
changes, even though they haven't changed anything.

Only one of `Default` or `DefaultFunc` may be set on any given field.

There is also an `InputDefault` property on your `Schema`. Unlike `Default` or
`DefaultFunc`, the user will still be prompted for input if `InputDefault` is
set and they don't specify a value. The value of `InputDefault` will just be
pre-populated in the input for the user.

Defaults can pose problems for backwards compatibility inside providers.
Because `Default` and the value provided by `DefaultFunc` are stored in state
and evaluated on every run, changing `Default` or the value provided by
`DefaultFunc` results in a diff for any user that has the previous value in
their state. Defaults can also pose problems when importing, as the default
will not be set on the imported Resource, and so the Resource winds up in a
state that it could not have been put in by a config, and an apply is necessary
before it can be fixed. When a field that has a default has
[`ForceNew`](#working-with-forcenew) set to `true` on it, as well, this is
compounded. [State migrations](#versioning-your-schema) can mitigate this, as
can the use of [`Computed`](#working-with-computed), but carefully considering
your defaults will pay off in terms of backwards compatibility for your users.

## Working With Sets and Lists

Sometimes your Provider needs to model fields that accept a group of values,
not just a single value. In this case, you want either a set or a list. A rule
of thumb is that if order is not important, you want a set. Otherwise, you want
a list. If in doubt, you probably want a list. One thing to know is that a set
can only have one of each value in it; if you want the same value to appear in
the group twice, you want a list.

To make a field accept a set, set its `Type` property to `schema.TypeSet`. The
`Set` property of the field can be set to a
[`schema.SchemaSetFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc)
to customize the way your values are hashed into keys, but unless you know you
need it, we recommend you just leave it empty and use the default hashing
function.

To make a field accept a list, set its `Type` property to `schema.TypeList`.

Regardless of whether the field accepts a list or a set, you need to also set
the `Elem` property to an instance of a `*schema.Schema` or a
`*schema.Resource`. A `*schema.Schema` will just be a simple value, but a
`*schema.Resource` houses a more complex structure, and could have its own
lifecycle for elements being added and removed.

When using a `*schema.Resource` as the `Elem` for a set, it's important to know
that fields on the `*schema.Resource` that are have their `Computed` property
set to `true` will cause problems. This should be avoided. You should also
avoid nesting a set inside of another set, as changing the contents of the
inner set will mark the outer set as changed, which leads to some
hard-to-understand diffs.

Also, keep in mind that it's hard to know the "key" for set items ahead of
time, so it will be hard to interpolate specific items of the set. Lists do not
have this problem, as the "key" is always their position in the list.

Note, however, that inserting a new item into the list at any point but the end
of the list will cause every item following the newly inserted item to think it
has "changed", even though only its position in the list has changed. When
you're able to update the entire list atomically, this isn't such a big deal,
but if updating the list involves making an API call for each element that has
changed, this can get messy.

If you're in doubt as to whether to use a list or a set, err on the side of the
list, as it's easier to switch from a list to a set than to switch from a set
to a list.

Finally, know that the optional [validation function](#customizing-validation)
you can supply will not work with lists or sets.

## Working With Either/Or Fields

There are times when a field in your resource conflicts with one or more other
fields, and only one should be set at a time. For example, a health check may
accept an HTTP endpoint, an HTTPS endpoint, or a TCP endpoint, but only one of
the three.

To help in this scenario, you can set your `Schema`'s `ConflictsWith` property.
Its value should be a list of the fields that the property conflicts with. It's
important to note that _each_ field must have _all_ the fields it conflicts
with set for this to work appropriately. For example, if A conflicts with B,
A's `ConflictsWith` property should be set to `"B"`, and `B`'s `ConflictsWith`
property should be set to `"A"`.

If the user sets a field that the given field has listed in its `ConflictsWith`
property, Terraform will return an error and exit execution.

## Customizing State

Sometimes it's not optimal to store the state _exactly_ as it's represented in
`*ResourceData`. For example, for large strings, it may be better to simply
store the hash of the string.

For these cases, the `StateFunc` property of your `Schema` can be used to
customise the way a field appears in state. This function will also be called,
with the config file's values as input, before comparing your state for
diffing.

To take advantage of this behaviour, set the `StateFunc` property to a
[`schema.SchemaStateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaStateFunc).
The interface will be the value that would be written to state, and the
`schema.SchemaStateFunc` needs to return the string it should be represented as
in state.

## Customizing Validation

While Terraform will do its best to ensure users are entering the right type of
inputs for your schema, its capabilities are pretty limited unless you give it
some information.

You must set either `Required`, `Optional`, or
[`Computed`](#working-with-computed) to `true`. You cannot set both `Required`
and `Optional` to `true`. If `Required` is `true`, Terraform will automatically
return an error if the field is not set. If `Optional` is `true`, Terraform
will not object to having an empty value (or no value at all) set for the
field. If only `Computed` is set to `true`, Terraform will return an error if
the user tries to enter any input at all for that field; this is most useful
for fields controlled entirely by the API.

For fields where you need even further control, your `Schema` has a
`ValidateFunc` property. This can be set to a
[`schema.SchemaValidateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaValidateFunc),
which allows you to define arbitrary validation logic and return either errors
(which will halt Terraform execution) or warnings (which will not halt
execution, but will be displayed in the output).

~> **Note:** `ValidateFunc` only works on primitive types; sets and lists
cannot use it for validation.

Validation is a bit of an art form. The stricter the validation you apply, the
sooner users will be notified of errors, and the more errors will be caught
before any resources are created. But you also run the risk of the API changing
its validation logic, and your provider calling valid input invalid,
frustrating users. A good rule of thumb is to apply validation when you believe
the API's validation is fairly static and unlikely to change.

## Customizing Diffs

For some forms of data, they may _look_ different, but are semantically the
same. In these cases, Terraform will think that something has changed, when in
reality, it hasn't. A great example of this is JSON data, which can be
represented any number of ways, but still mean the same thing.

For these cases, it's a bad user experience to ignore semantic equality and let
Terraform just enforce lexicographic equality, especially if the API isn't
consistent about its output format, or if the field in question has
[`ForceNew`](#working-with-forcenew) set to `true`.

Instead, use the `DiffSuppressFunc` property of your `Schema` to supply a
[`schema.SchemaDiffSuppressFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaDiffSuppressFunc)
to help Terraform understand which values should be considered equal. The
function will be called with the key of the field in question, the value in
state, the new value, and the `*ResourceData` the new value came from. It
should return `true` if the values should be considered equal (and therefore,
not a diff), or false if the values should be considered unequal (and
therefore, a diff).

~> **Note:** `DiffSuppressFunc` should only be used on primitive types; sets
and lists will be challenging to use it correctly with.

## Storing Sensitive Information

Terraform's state may sometimes hold sensitive information. For these fields,
you don't want them just displayed in all the diffs or to anyone that runs
`terraform show`. To prevent fields from being shown in output, set the
`Sensitive` property of their `Schema` to `true`.

~> **Note:** This _does not_ protect the information in state. In the future it
may, but for now, it's important to know how to [protect your
statefiles](/docs/state/sensitive-data.html).

## Versioning Your Schema

Maintaining backwards compatibility is an important part of maintaining a
Terraform provider. Because Terraform manages infrastructure, it can be
expensive and disruptive to have a breaking change.

To help with this, the state associated with each of your resources is
versioned, and you can increment the versions and tell Terraform how to migrate
the state of existing users. This allows you to change the shape of the state
you're managing over time while automating away the upgrade process for users.

This generally is only needed when you're changing the way an existing field is
persisted in state; new fields almost always work without a migration, and
removing fields almost always works without a migration. It should be fairly
straightforward to test if a migraiton is necesary: set up state using the
previous version of your provider, then upgrade to the new version, and try
using `terraform plan` or `terraform apply`. If you see any unexpected diffs or
errors, you may need a state migration.

To use the migration functionality, increment the `SchemaVersion` property of
your `Resource`. If it's not set, set it to `1`.

The `MigrateState` property is where you'll set a
[`schema.StateMigrateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#StateMigrateFunc),
which allows you to define arbitrary logic for migrating the state of the
`Resource`.

The `schema.StateMigrateFunc` will be passed the version of the schema the
state is currently in, a pointer to the
[`terraform.InstanceState`](https://godoc.org/github.com/hashicorp/terraform/terraform#InstanceState)
that needs to be migrated to the new format, and the `interface{}` returned by
your provider's
[`ConfigureFunc`](/docs/internals/providers/new-provider.html#configuring-your-provider).

The version the state is in is important, as your migration function should
allow users of _any_ previous state (not just the most recent one) to migrate
to the new format.

The `terraform.InstanceState` will allow you to inspect and set values (using
its `Attributes` property, which is a map with the keys being the field and the
values being an `interface{}` representation of the field's value).

The `interface{}` returned by your provider's `ConfigureFunc` is useful in case
you need to make API calls to properly migrate the state.

Your function should return the new `*terraform.InstanceState`, or an `error`
if the state could not be migrated.

This function will be run during the refresh portion of the Terraform
lifecycle, and so should get executed before any of your provider's lifecycle
functions are called.
