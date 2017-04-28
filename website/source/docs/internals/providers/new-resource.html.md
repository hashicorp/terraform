---
layout: "docs"
page_title: "Creating Resources"
sidebar_current: "docs-internals-provider-guide-new-resource"
description: |-
  How to get started adding a new resource to an existing provider.
---

# Creating Resources

Now that you've got a Provider, it's time to add some Resources to that
Provider to make it actually _do_ something. To do this, you need to write some
code in the package your Provider is defined in.

In the built-in plugins for Terraform that are shipped as part of the codebase,
we have a file naming guideline: we use `resource_{resource_name}.go` as the
template to name our files, and tend to stick to  single Resource per file. For
example, AWS has `resource_aws_instance.go`, because it uses the AWS API to
manage the instance resource. If you want to contribute your code back to the
Terraform repo, it makes things easier if you follow this template.

Inside this file, you're going to define a function that takes no arguments and
returns a `*schema.Resource`.  While you can technically name your function
anything, we recommend (especially if you plan on contributing your code back
to the Terraform repo!) that you name it the camelCase version of the file
name. For example, `resource_compute_instance.go` would contain
`resourceComputeInstance`. We'll be defining a _lot_ of functions in a
Provider, and this kind of naming scheme really helps keep things organised and
easy to find.

## Registering the Resource With the Provider

Now that you've got your function, switch back to your Provider definition, and
add the Resource to the Provider's `ResourcesMap` property. The key will be the
name of the Resource used in state and configuration files, and (by convention)
should match the `{resource_name}` part used in the filename for the file that
defines the Resource. Nothing will necessarily break if you don't do this, but
if you happen to accidentally have a collision with any other resource in any
other provider, things will likely break. This convention helps keep things
neat and orderly while avoiding conflicts.

## Defining the Resource Properties

Each Resource has a set of properties called the "schema" that is stored in the
state. Think of it as the type definition for the Resource. For example, an AWS
instance resource has a `name` property, to control the name of the instance,
GitHub's repository resource has a `default_branch` property to set the default
branch for the repository, and so on.

These properties are defined in the `Schema` property of the Resource
definition. It takes a map with the keys being the property name and the values
being `*Schema` structs. The `*Schema` structs define some type information
(what kind of data to expect, etc.) along with some [advanced
behaviour](/docs/internals/providers/schema.html) for resources that helps
Terraform do the right thing without you needing to write a bunch of code.

~> **Note:** "id" is a reserved property name. Don't call your property "id".

## Calling the API

Now that we have a Resource fully defined, it's time to make it do something.
We're going to use the Provider's API client (possibly
[configured](/docs/internals/providers/new-provider.html#configuring-your-provider)
by the Provider's `ConfigureFunc`) to create, read, update, destroy, and check
for the existence of some resources. We do this by defining functions to
perform those operations for us. Terraform will then decide which of the
functions to call and on which resources. All the functions have to do is know
how to determine and set the state of a resource using the Provider's API.

Each function takes a `*ResourceData` struct and an `interface{}` as arguments,
and returns an `error`. The only exception is `ExistsFunc`, which takes the
same arguments, but returns a boolean or an `error`.

The `*ResourceData` struct contains the state of the Resource as it should be.
It's an amalgamation of several different sources of data, which are explained
further in [Using ResourceData](/docs/internals/providers/resource-data.html).

The `interface{}` is the same `interface{}` returned by the Provider's
`ConfigureFunc`. Typically, this is where you'd put the configured API client,
for example.

Of all these functions, only the `Update` and `Exists` functions are optional.
If the `Update` function isn't set, the Resource will be treated as something
that cannot be updated. If the `Exists` function isn't set, Terraform will not
check if the Resource already/still exists.

The `Create` function should use the passed `*ResourceData` to make an API call
that will create a Resource in the desired state. It should then update the
`*ResourceData` with the state as reported by the server, to detect any drift
and fill in any `Computed` properties.

The `Read` function should  retrieve the ID (or other identifying information)
of the resource to be read from the passed `*ResourceData`. It should then
retrieve the Resource from the API, and set the `*ResourceData`'s fields to
match what the server reported. It's important to note that even if it's
required to set state as user input, the `Read` function should still overwrite
that state, so Terraform can understand the current state of the world, not
just the desired state.

The `Update` function is called when the state of one of your resources has
diverged from the config file, but the resource already exists. If the API
supports partial updates of resources (for example, using `PATCH` requests)
your update function  can use the passed `*ResourceData`'s `HasChange` method
to detect which properties need to be updated, and make the necessary API
calls; otherwise, use the `*ResourceData`'s `Get` methods to construct your
request, just like when creating a resource. Terraform makes no effort to roll
back failed updates, so if your `Update` function makes multiple API calls or
performs other half-updates, you should make sure to [set partial state
mode](/docs/internals/providers/resource-data.html#partial-updates) on the
`*ResourceData` and use it carefully.

The `Delete` function should use the passed `*ResourceData` to retrieve the ID
(or other identifying information necessary for the API call) of the Resource
to be deleted, then make the API call or calls necessary to delete that
Resource. Sometimes APIs can be picky about the order Resources are deleted in;
Terraform always deletes resources that depend on other resources before
deleting the depended upon resources.

The `Exists` function should use the passed `*ResourceData` containing the ID
(or other identifying information necessary for the API call) of the Resource
to check the existence of, then make the API call necessary to check for its
existence. If the Resource exists, the `Exists` function should return true.
It's important that the `Exists` function never modify the passed
`*ResourceData`. It's also important that the `Exists` function always returns
errors it runs into while checking whether the Resource exists or not;
Resources should only be treated as "gone" if they can't be found, not if
there's a failed API call.
