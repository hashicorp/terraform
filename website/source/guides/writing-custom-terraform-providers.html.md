---
layout: "guides"
page_title: "Writing Custom Providers - Guides"
sidebar_current: "guides-writing-custom-terraform-providers"
description: |-
  Terraform providers are easy to create and manage. This guide demonstrates
  authoring a Terraform provider from scratch.
---

# Writing Custom Providers

~> **This is an advanced guide!** Following this guide is not required for
regular use of Terraform and is only intended for advance users or Terraform
contributors.

In Terraform, a "provider" is the logical abstraction of an upstream API. This
guide details how to build a custom provider for Terraform.

## Why?

There are a few possible reasons for authoring a custom Terraform provider, such
as:

- An internal private cloud whose functionality is either proprietary or would
  not benefit the open source community.

- A "work in progress" provider being tested locally before contributing back.

- Extensions of an existing provider

## Local Setup

Terraform supports a plugin model, and all providers are actually plugins.
Plugins are distributed as Go binaries. Although technically possible to write a
plugin in another language, almost all Terraform plugins are written in
[Go](https://golang.org). For more information on installing and configuring Go,
please visit the [Golang installation guide](https://golang.org/doc/install).

This post assumes familiarity with Golang and basic programming concepts.

As a reminder, all of Terraform's core providers are open source. When stuck or
looking for examples, please feel free to reference
[the open source providers](https://github.com/hashicorp/terraform/tree/master/builtin/providers) for help.

## The Provider Schema

To start, create a file named `provider.go`. This is the root of the provider
and should include the following boilerplate code:

```go
package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{},
	}
}
```

The
[`helper/schema`](https://godoc.org/github.com/hashicorp/terraform/helper/schema)
library is part of Terraform's core. It abstracts many of the complexities and
ensures consistency between providers. The example above defines an empty provider (there are no _resources_).

The `*schema.Provider` type describes the provider's properties including:

- the configuration keys it accepts
- the resources it supports
- any callbacks to configure

## Building the Plugin

Go requires a `main.go` file, which is the default executable when the binary is
built. Since Terraform plugins are distributed as Go binaries, it is important
to define this entry-point with the following code:

```go
package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return Provider()
		},
	})
}
```

This establishes the main function to produce a valid, executable Go binary. The
contents of the main function consume Terraform's `plugin` library. This library
deals with all the communication between Terraform core and the plugin.

Next, build the plugin using the Go toolchain:

```shell
$ go build -o terraform-provider-example
```

The output name (`-o`) is **very important**. Terraform searches for plugins in
the format of:

```text
terraform-<TYPE>-<NAME>
```

In the case above, the plugin is of type "provider" and of name "example".

To verify things are working correctly, execute the binary just created:

```shell
$ ./terraform-provider-example
This binary is a plugin. These are not meant to be executed directly.
Please execute the program that consumes these plugins, which will
load any plugins automatically
```

This is the basic project structure and scaffolding for a Terraform plugin. To
recap, the file structure is:

```text
.
├── main.go
└── provider.go
```

## Defining Resources

Terraform providers manage resources. A provider is an abstraction of an
upstream API, and a resource is a component of that provider. As an example, the
AWS provider supports `aws_instance` and `aws_elastic_ip`. DNSimple supports
`dnsimple_record`. Fastly supports `fastly_service`. Let's add a resource to our
fictitious provider.

As a general convention, Terraform providers put each resource in their own
file, named after the resource, prefixed with `resource_`. To create an
`example_server`, this would be `resource_server.go` by convention:

```go
package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceServerCreate,
		Read:   resourceServerRead,
		Update: resourceServerUpdate,
		Delete: resourceServerDelete,

		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

```

This uses the
[`schema.Resource` type](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Resource).
This structure defines the data schema and CRUD operations for the resource.
Defining these properties are the only required thing to create a resource.

The schema above defines one element, `"address"`, which is a required string.
Terraform's schema automatically enforces validation and type casting.

Next there are four "fields" defined - `Create`, `Read`, `Update`, and `Delete`.
The `Create`, `Read`, and `Delete` functions are required for a resource to be
functional. There are other functions, but these are the only required ones.
Terraform itself handles which function to call and with what data. Based on the
schema and current state of the resource, Terraform can determine whether it
needs to create a new resource, update an existing one, or destroy.

Each of the four struct fields point to a function. While it is technically
possible to inline all functions in the resource schema, best practice dictates
pulling each function into its own method. This optimizes for both testing and
readability. Fill in those stubs now, paying close attention to method
signatures.

```golang
func resourceServerCreate(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceServerRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceServerUpdate(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceServerDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}
```

Lastly, update the provider schema in `provider.go` to register this new resource.

```golang
func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"example_server": resourceServer(),
		},
	}
}
```

Build and test the plugin. Everything should compile as-is, although all
operations are a no-op.

```shell
$ go build -o terraform-provider-example

$ ./terraform-provider-example
This binary is a plugin. These are not meant to be executed directly.
Please execute the program that consumes these plugins, which will
load any plugins automatically
```

The layout now looks like this:

```text
.
├── main.go
├── provider.go
├── resource_server.go
└── terraform-provider-example
```

## Invoking the Provider

Previous sections showed running the provider directly via the shell, which
outputs a warning message like:

```text
This binary is a plugin. These are not meant to be executed directly.
Please execute the program that consumes these plugins, which will
load any plugins automatically
```

Terraform plugins should be executed by Terraform directly. To test this, create
a `main.tf` in the working directory (the same place where the plugin exists).

```hcl
resource "example_server" "my-server" {}
```

And execute `terraform plan`:

```text
$ terraform plan

1 error(s) occurred:

* example_server.my-server: "address": required field is not set
```

This validates Terraform is correctly delegating work to our plugin and that our
validation is working as intended. Fix the validation error by adding an
`address` field to the resource:

```hcl
resource "example_server" "my-server" {
  address = "1.2.3.4"
}
```

Execute `terraform plan` to verify the validation is passing:

```text
$ terraform plan

+ example_server.my-server
    address: "1.2.3.4"


Plan: 1 to add, 0 to change, 0 to destroy.
```

It is possible to run `terraform apply`, but it will be a no-op because all of
the resource options currently take no action.

## Implement Create

Back in `resource_server.go`, implement the create functionality:

```go
func resourceServerCreate(d *schema.ResourceData, m interface{}) error {
	address := d.Get("address").(string)
	d.SetId(address)
	return nil
}
```

This uses the [`schema.ResourceData
API`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ResourceData)
to get the value of `"address"` provided by the user in the Terraform
configuration. Due to the way Go works, we have to typecast it to string. This
is a safe operation, however, since our schema guarantees it will be a string
type.

Next, it uses `SetId`, a built-in function, to set the ID of the resource to the
address. The existence of a non-blank ID is what tells Terraform that a resource
was created. This ID can be any string value, but should be a value that can be
used to read the resource again.

Recompile the binary, the run `terraform plan` and `terraform apply`.

```shell
$ go build -o terraform-provider-example
# ...
```

```text
$ terraform plan

+ example_server.my-server
    address: "1.2.3.4"


Plan: 1 to add, 0 to change, 0 to destroy.
```

```text
$ terraform apply

example_server.my-server: Creating...
  address: "" => "1.2.3.4"
example_server.my-server: Creation complete (ID: 1.2.3.4)

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```

Since the `Create` operation used `SetId`, Terraform believes the resource created successfully. Verify this by running `terraform plan`.

```text
$ terraform plan
Refreshing Terraform state in-memory prior to plan...
The refreshed state will be used to calculate this plan, but will not be
persisted to local or remote state storage.

example_server.my-server: Refreshing state... (ID: 1.2.3.4)
No changes. Infrastructure is up-to-date.

This means that Terraform did not detect any differences between your
configuration and real physical resources that exist. As a result, Terraform
doesn't need to do anything.
```

Again, because of the call to `SetId`, Terraform believes the resource was
created. When running `plan`, Terraform properly determines there are no changes
to apply.

To verify this behavior, change the value of the `address` field and run
`terraform plan` again. You should see output like this:

```text
$ terraform plan
example_server.my-server: Refreshing state... (ID: 1.2.3.4)

~ example_server.my-server
    address: "1.2.3.4" => "5.6.7.8"


Plan: 0 to add, 1 to change, 0 to destroy.
```

Terraform detects the change and displays a diff with a `~` prefix, noting the
resource will be modified in place, rather than created new.

Run `terraform apply` to apply the changes.

```text
$ terraform apply
example_server.my-server: Refreshing state... (ID: 1.2.3.4)
example_server.my-server: Modifying... (ID: 1.2.3.4)
  address: "1.2.3.4" => "5.6.7.8"
example_server.my-server: Modifications complete (ID: 1.2.3.4)

Apply complete! Resources: 0 added, 1 changed, 0 destroyed.
```

Since we did not implement the `Update` function, you would expect the
`terraform plan` operation to report changes, but it does not! How were our
changes persisted without the `Update` implementation?

## Error Handling &amp; Partial State

Previously our `Update` operation succeeded and persisted the new state with an
empty function definition. Recall the current update function:

```golang
func resourceServerUpdate(d *schema.ResourceData, m interface{}) error {
	return nil
}
```

The `return nil` tells Terraform that the update operation succeeded without
error. Terraform assumes this means any changes requested applied without error.
Because of this, our state updated and Terraform believes there are no further
changes.

To say it another way: if a callback returns no error, Terraform automatically
assumes the entire diff successfully applied, merges the diff into the final
state, and persists it.

Functions should _never_ intentionally `panic` or call `os.Exit` - always return
an error.

In reality, it is a bit more complicated than this. Imagine the scenario where
our update function has to update two separate fields which require two separate
API calls. What do we do if the first API call succeeds but the second fails?
How do we properly tell Terraform to only persist half the diff? This is known
as a _partial state_ scenario, and implementing these properly is critical to a
well-behaving provider.

Here are the rules for state updating in Terraform. Note that this mentions
callbacks we have not discussed, for the sake of completeness.

- If the `Create` callback returns with or without an error without an ID set
  using `SetId`, the resource is assumed to not be created, and no state is
  saved.

- If the `Create` callback returns with or without an error and an ID has been
  set, the resource is assumed created and all state is saved with it. Repeating
  because it is important: if there is an error, but the ID is set, the state is
  fully saved.

- If the `Update` callback returns with or without an error, the full state is
  saved. If the ID becomes blank, the resource is destroyed (even within an
  update, though this shouldn't happen except in error scenarios).

- If the `Destroy` callback returns without an error, the resource is assumed to
  be destroyed, and all state is removed.

- If the `Destroy` callback returns with an error, the resource is assumed to
  still exist, and all prior state is preserved.

- If partial mode (covered next) is enabled when a create or update returns,
  only the explicitly enabled configuration keys are persisted, resulting in a
  partial state.

_Partial mode_ is a mode that can be enabled by a callback that tells Terraform
that it is possible for partial state to occur. When this mode is enabled, the
provider must explicitly tell Terraform what is safe to persist and what is not.

Here is an example of a partial mode with an update function:

```go
func resourceServerUpdate(d *schema.ResourceData, m interface{}) error {
	// Enable partial state mode
	d.Partial(true)

	if d.HasChange("address") {
		// Try updating the address
		if err := updateAddress(d, m); err != nil {
			return err
		}

		d.SetPartial("address")
	}

	// If we were to return here, before disabling partial mode below,
	// then only the "address" field would be saved.

	// We succeeded, disable partial mode. This causes Terraform to save
	// save all fields again.
	d.Partial(false)

	return nil
}
```

Note - this code will not compile since there is no `updateAddress` function.
You can implement a dummy version of this function to play around with partial
state. For this example, partial state does not mean much in this documentation
example. If `updateAddress` were to fail, then the address field would not be
updated.

## Implementing Destroy

The `Destroy` callback is exactly what it sounds like - it is called to destroy
the resource. This operation should never update any state on the resource. It
is not necessary to call `d.SetId("")`, since any non-error return value assumes
the resource was deleted successfully.

```go
func resourceServerDelete(d *schema.ResourceData, m interface{}) error {
  // d.SetId("") is automatically called assuming delete returns no errors, but
  // it is added here for explicitness.
	d.SetId("")
	return nil
}
```

The destroy function should always handle the case where the resource might
already be destroyed (manually, for example). If the resource is already
destroyed, this should not return an error. This allows Terraform users to
manually delete resources without breaking Terraform.

```shell
$ go build -o terraform-provider-example
```

Run `terraform destroy` to destroy the resource.

```text
$ terraform destroy
Do you really want to destroy?
  Terraform will delete all your managed infrastructure.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value: yes

example_server.my-server: Refreshing state... (ID: 5.6.7.8)
example_server.my-server: Destroying... (ID: 5.6.7.8)
example_server.my-server: Destruction complete

Destroy complete! Resources: 1 destroyed.
```

## Implementing Read

The `Read` callback is used to sync the local state with the actual state
(upstream). This is called at various points by Terraform and should be a
read-only operation. This callback should never modify the real resource.

If the ID is updated to blank, this tells Terraform the resource no longer
exists (maybe it was destroyed out of band). Just like the destroy callback, the
`Read` function should gracefully handle this case.

```go
func resourceServerRead(d *schema.ResourceData, m interface{}) error {
  client := m.(*MyClient)

  // Attempt to read from an upstream API
  obj, ok := client.Get(d.Id())

  // If the resource does not exist, inform Terraform. We want to immediately
  // return here to prevent further processing.
  if !ok {
    d.SetId("")
    return nil
  }

  d.Set("address", obj.Address)
  return nil
}
```

## Next Steps

This guide covers the schema and structure for implementing a Terraform provider
using the provider framework. As next steps, reference the internal providers
for examples. Terraform also includes a full framework for testing frameworks.

## General Rules

### Dedicated Upstream Libraries

One of the biggest mistakes new users make is trying to conflate a client
library with the Terraform implementation. Terraform should always consume an
independent client library which implements the core logic for communicating
with the upstream. Do not try to implement this type of logic in the provider
itself.

### Data Sources

While not explicitly discussed here, _data sources_ are a special subset of
resources which are read-only. They are resolved earlier than regular resources
and can be used as part of Terraform's interpolation.
