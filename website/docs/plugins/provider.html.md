---
layout: "docs"
page_title: "Provider Plugins"
sidebar_current: "docs-plugins-provider"
description: |-
  A provider in Terraform is responsible for the lifecycle of a resource: create, read, update, delete. An example of a provider is AWS, which can manage resources of type `aws_instance`, `aws_eip`, `aws_elb`, etc.
---

# Provider Plugins

A provider in Terraform is responsible for the lifecycle of a resource:
create, read, update, delete. An example of a provider is AWS, which
can manage resources of type `aws_instance`, `aws_eip`, `aws_elb`, etc.

The primary reasons to care about provider plugins are:

  * You want to add a new resource type to an existing provider.

  * You want to write a completely new provider for managing resource
    types in a system not yet supported.

  * You want to write a completely new provider for custom, internal
    systems such as a private inventory management system.

~> **Advanced topic!** Plugin development is a highly advanced
topic in Terraform, and is not required knowledge for day-to-day usage.
If you don't plan on writing any plugins, we recommend not reading
this section of the documentation.

If you're interested in provider development, then read on. The remainder
of this page will assume you're familiar with
[plugin basics](/docs/plugins/basics.html) and that you already have
a basic development environment setup.

## Provider Plugin Codebases

Provider plugins live outside of the Terraform core codebase in their own
source code repositories. The official set of provider plugins released by
HashiCorp (developed by both HashiCorp staff and community contributors)
all live in repositories in
[the `terraform-providers` organization](https://github.com/terraform-providers)
on GitHub, but third-party plugins can be maintained in any source code
repository.

When developing a provider plugin, it is recommended to use a common `GOPATH`
that includes both the core Terraform repository and the repositories of any
providers being changed. This makes it easier to use a locally-built
`terraform` executable and a set of locally-built provider plugins together
without further configuration.

For example, to download both Terraform and the `template` provider into
`GOPATH`:

```
$ go get github.com/hashicorp/terraform
$ go get github.com/terraform-providers/terraform-provider-template
```

These two packages are both "main" packages that can be built into separate
executables with `go install`:

```
$ go install github.com/hashicorp/terraform
$ go install github.com/terraform-providers/terraform-provider-template
```

After running the above commands, both Terraform core and the `template`
provider will both be installed in the current `GOPATH` and `$GOPATH/bin`
will contain both `terraform` and `terraform-provider-template` executables.
This `terraform` executable will find and use the `template` provider plugin
alongside it in the `bin` directory in preference to downloading and installing
an official release.

When constructing a new provider from scratch, it's recommended to follow
a similar repository structure as for the existing providers, with the main
package in the repository root and a library package in a subdirectory named
after the provider. For more information, see
[the custom providers guide](/guides/writing-custom-terraform-providers.html).

When making changes only to files within the provider repository, it is _not_
necessary to re-build the main Terraform executable. Note that some packages
from the Terraform repository are used as library dependencies by providers,
such as `github.com/hashicorp/terraform/helper/schema`; it is recommended to
use `govendor` to create a local vendor copy of the relevant packages in the
provider repository, as can be seen in the repositories within the
`terraform-providers` GitHub organization.

## Low-Level Interface

The interface you must implement for providers is
[ResourceProvider](https://github.com/hashicorp/terraform/blob/master/terraform/resource_provider.go).

This interface is extremely low level, however, and we don't recommend
you implement it directly. Implementing the interface directly is error
prone, complicated, and difficult.

Instead, we've developed some higher level libraries to help you out
with developing providers. These are the same libraries we use in our
own core providers.

## helper/schema

The `helper/schema` library is a framework we've built to make creating
providers extremely easy. This is the same library we use to build most
of the core providers.

To give you an idea of how productive you can become with this framework:
we implemented the Google Cloud provider in about 6 hours of coding work.
This isn't a simple provider, and we did have knowledge of
the framework beforehand, but it goes to show how expressive the framework
can be.

The GoDoc for `helper/schema` can be
[found here](https://godoc.org/github.com/hashicorp/terraform/helper/schema).
This is API-level documentation but will be extremely important
for you going forward.

## Provider

The first thing to do in your plugin is to create the
[schema.Provider](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Provider) structure.
This structure implements the `ResourceProvider` interface. We
recommend creating this structure in a function to make testing easier
later. Example:

```golang
func Provider() *schema.Provider {
	return &schema.Provider{
		...
	}
}
```

Within the `schema.Provider`, you should initialize all the fields. They
are documented within the godoc, but a brief overview is here as well:

  * `Schema` - This is the configuration schema for the provider itself.
      You should define any API keys, etc. here. Schemas are covered below.

  * `ResourcesMap` - The map of resources that this provider supports.
      All keys are resource names and the values are the
      [schema.Resource](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Resource) structures implementing this resource.

  * `ConfigureFunc` - This function callback is used to configure the
      provider. This function should do things such as initialize any API
      clients, validate API keys, etc. The `interface{}` return value of
      this function is the `meta` parameter that will be passed into all
      resource [CRUD](https://en.wikipedia.org/wiki/Create,_read,_update_and_delete)
      functions. In general, the returned value is a configuration structure
      or a client.

As part of the unit tests, you should call `InternalValidate`. This is used
to verify the structure of the provider and all of the resources, and reports
an error if it is invalid. An example test is shown below:

```golang
func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
```

Having this unit test will catch a lot of beginner mistakes as you build
your provider.

## Resources

Next, you'll want to create the resources that the provider can manage.
These resources are put into the `ResourcesMap` field of the provider
structure. Again, we recommend creating functions to instantiate these.
An example is shown below.

```golang
func resourceComputeAddress() *schema.Resource {
	return &schema.Resource {
		...
	}
}
```

Resources are described using the
[schema.Resource](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Resource)
structure. This structure has the following fields:

  * `Schema` - The configuration schema for this resource. Schemas are
      covered in more detail below.

  * `Create`, `Read`, `Update`, and `Delete` - These are the callback
      functions that implement CRUD operations for the resource. The only
      optional field is `Update`. If your resource doesn't support update, then
      you may keep that field nil.

  * `Importer` - If this is non-nil, then this resource is
    [importable](/docs/import/importability.html). It is recommended to
    implement this.

The CRUD operations in more detail, along with their contracts:

  * `Create` - This is called to create a new instance of the resource.
      Terraform guarantees that an existing ID is not set on the resource
      data. That is, you're working with a new resource. Therefore, you are
      responsible for calling `SetId` on your `schema.ResourceData` using a
      value suitable for your resource. This ensures whatever resource
      state you set on `schema.ResourceData` will be persisted in local state.
      If you neglect to `SetId`, no resource state will be persisted.

  * `Read` - This is called to resync the local state with the remote state.
      Terraform guarantees that an existing ID will be set. This ID should be
      used to look up the resource. Any remote data should be updated into
      the local data. **No changes to the remote resource are to be made.**

  * `Update` - This is called to update properties of an existing resource.
      Terraform guarantees that an existing ID will be set. Additionally,
      the only changed attributes are guaranteed to be those that support
      update, as specified by the schema. Be careful to read about partial
      states below.

  * `Delete` - This is called to delete the resource. Terraform guarantees
      an existing ID will be set.

  * `Exists` - This is called to verify a resource still exists. It is
      called prior to `Read`, and lowers the burden of `Read` to be able
      to assume the resource exists. If the resource is no longer present in
      remote state,  calling `SetId` with an empty string will signal its removal.

## Schemas

Both providers and resources require a schema to be specified. The schema
is used to define the structure of the configuration, the types, etc. It is
very important to get correct.

In both provider and resource, the schema is a `map[string]*schema.Schema`.
The key of this map is the configuration key, and the value is a schema for
the value of that key.

Schemas are incredibly powerful, so this documentation page won't attempt
to cover the full power of them. Instead, the API docs should be referenced
which cover all available settings.

We recommend viewing schemas of existing or similar providers to learn
best practices. A good starting place is the
[core Terraform providers](https://github.com/hashicorp/terraform/tree/master/builtin/providers).

## Resource Data

The parameter to provider configuration as well as all the CRUD operations
on a resource is a
[schema.ResourceData](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ResourceData).
This structure is used to query configurations as well as to set information
about the resource such as its ID, connection information, and computed
attributes.

The API documentation covers ResourceData well, as well as the core providers
in Terraform.

**Partial state** deserves a special mention. Occasionally in Terraform, create or
update operations are not atomic; they can fail halfway through. As an example,
when creating an AWS security group, creating the group may succeed,
but creating all the initial rules may fail. In this case, it is incredibly
important that Terraform record the correct _partial state_ so that a
subsequent `terraform apply` fixes this resource.

Most of the time, partial state is not required. When it is, it must be
specifically enabled. An example is shown below:

```golang
func resourceUpdate(d *schema.ResourceData, meta interface{}) error {
	// Enable partial state mode
	d.Partial(true)

	if d.HasChange("tags") {
		// If an error occurs, return with an error,
		// we didn't finish updating
		if err := updateTags(d, meta); err != nil {
			return err
		}

		d.SetPartial("tags")
	}

	if d.HasChange("name") {
		if err := updateName(d, meta); err != nil {
			return err
		}

		d.SetPartial("name")
	}

	// We succeeded, disable partial mode
	d.Partial(false)

	return nil
}
```

In the example above, it is possible that setting the `tags` succeeds,
but setting the `name` fails. In this scenario, we want to make sure
that only the state of the `tags` is updated. To do this the
`Partial` and `SetPartial` functions are used.

`Partial` toggles partial-state mode. When disabled, all changes are merged
into the state upon result of the operation. When enabled, only changes
enabled with `SetPartial` are merged in.

`SetPartial` tells Terraform what state changes to adopt upon completion
of an operation. You should call `SetPartial` with every key that is safe
to merge into the state. The parameter to `SetPartial` is a prefix, so
if you have a nested structure and want to accept the whole thing,
you can just specify the prefix.
