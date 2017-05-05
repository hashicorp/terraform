---
layout: "docs"
page_title: "Creating Resources"
sidebar_current: "docs-internals-provider-guide-new-resource"
description: |-
  How to get started adding a new resource to an existing provider.
---

# Creating Resources

Resources are the conceptual building blocks of Terraform. They are what Terraform
manipulates, the pieces of infrastructure being codified.

Resources are configured as a map on a [`schema.Provider`](new-provider.html)'s
`ResourcesMap` property. The key is the field name that will be used in
configuration files. The value is a [`*schema.Resource`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Resource).

Most providers define each `*schema.Resource` in its own file, as a function
Most providers have a file per resource. The file contains a function that takes
no arguments and returns a `*schema.Resource`. For example, to create a `bar` resource
in provider `foo`, `provider.go` contains:

```go
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"foo_bar": resourceBar(),
		},
	}
}
```

And `resource_bar.go` contains:

```go
func resourceBar() *schema.Resource{
	return &schema.Resource{
	}
}
```

This resource would then be configured with the following HCL:

```hcl
resource "foo_bar" "baz" {
}
```

The providers built into Terraform all use this `resource_<RESOURCE_NAME>.go` file
naming convention and contain a single resource per file. They also use the `resource<RESOURCE_NAME>`
naming convention for the functions that return `*schema.Resource`s. These conventions are not required,
but are recommended.

~> **Note:** Keys for the `ResourceMap` must be unique across _all_ providers.
Prefixing them with the provider name is _highly_ recommended.

## Defining the Resource Fields

Each Resource has a set of fields called the "schema" that is stored in the
state. The schema is essentially the type definition for the resource. For
example, an AWS instance resource has a `name` field, to control the name of
the instance, GitHub's repository resource has a `default_branch` field to set
the default branch for the repository, and so on.

The `Schema` property of the `*schema.Resource` defines these fields. It takes
a map with the field names as keys and `*schema.Schema`s as values. The
`*schema.Schema`s define type information (what kind of data to expect, etc.)
and [advanced behavior](schema.html) for the fields that helps Terraform do the
right thing without providers needing a bunch of code.

~> **Note:** "id" is a reserved field name. Fields must not be named "id".

To continue the example above, the `foo_bar` resource may have the following
`Schema` property:

```go
func resourceBar() *schema.Resource{
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"quux": {
				Type: TypeString,
				Required: true,
				ForceNew: true,
			},
			"corge": {
				Type: TypeBool,
				Optional: true,
			},
		},
	}
}
```

The HCL configuration to use it looks like this:

```hcl
resource "foo_bar" "baz" {
	quux = "abc123"
	corge = true
}
```

## Calling the API

Resources define their behavior through a set of functions that Terraform
calls at the appropriate times. These functions are set as properties on the
`*schema.Resource`.

### Creating Resources

The `Create` property should be set to a
[`schema.CreateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#CreateFunc),
which will be called whenever Terraform detects the resource is defined in HCL
but is not set in state. The `schema.CreateFunc` receives a
[`*schema.ResourceData`](resource-data.html) and the `interface{}` returned
by the provider's
[`ConfigureFunc`](new-provider.html#instantiating-clients). The function
should read the information it needs to create the resource from the
[`*schema.ResourceData`](resource-data.html) and make whatever API calls it
needs to to create the resource described. If the resource is successfully
created, the function should [update the
`*schema.ResourceData`](resource-data.html#setting-state) with the state as
reported by the API and return `nil`. If an error is encountered, the function
should return an `error` describing the problem, which will be surfaced to the
user. The `*schema.ResourceData` should reflect the API's output, to fill in
any [`Computed` fields](schema.html#working-with-computed) and detect any drift.

```go
func resourceBarCreate(d *schema.ResourceData, meta interface{}) error {
	// retrieve the information from the *schema.ResourceData
	quux := d.Get("quux").(string)
	corge, corgeSet := d.GetOk("corge")

	// cast meta to the client we returned from the ConfigureFunc
	client := meta.(fooClient)
	
	// build our request -- this will look different for each provider
	req := client.NewBar(quux)
	if corgeSet {
		req.Corge = corge.(bool)
	}

	// make our API request
	resp, err := req.Execute()
	if err != nil {
		// return an error
		return fmt.Sprintf("Error creating bar: %s", err.Error())
	}
	
	// update the *schema.ResourceData
	// most providers do this by immediately calling through to the schema.ReadFunc
	d.SetId(resp.Quux) // "quux" is the resource ID
	d.Set("corge", resp.Corge)
	return nil
}
```

### Reading Resources

The `Read` property should be set to a
[`schema.ReadFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ReadFunc),
which will be called whenever Terraform needs to synchronize its internal state
with the state reported by the API. The `schema.ReadFunc` receives a
[`*schema.ResourceData`](resource-data.html) and the `interface{}` returned
by the provider's [`ConfigureFunc`](new-provider.html#instantiating-clients).
The function should read the information it needs to retrieve the resource from
the [`*schema.ResourceData`](resource-data.html) and make whatever API calls it
needs to retrieve the resource described. If the resource is successfully
retrieved, the function should [update the
`*schema.ResourceData`](resource-data.html#setting-state) with the state as
reported by the API and return `nil`. If an error is encountered, the function
should return an `error` describing the problem, which will be surfaced to the
user. The `*schema.ResourceData` should reflect the API's output, to fill in
any [`Computed` fields](schema.html#working-with-computed) and detect any drift.

```go
func resourceBarRead(d *schema.ResourceData, meta interface{}) error {
	// retrieve the information from the *schema.ResourceData
	quux := d.GetId().(string)

	// cast meta to the client we returned from the ConfigureFunc
	client := meta.(fooClient)
	
	// build our request -- this will look different for each provider
	req := client.GetBar(quux)

	// make our API request
	resp, err := req.Execute()
	if err != nil {
		// return an error
		return fmt.Sprintf("Error retrieving bar: %s", err.Error())
	}
	
	// update the *schema.ResourceData
	d.SetId(resp.Quux) // "quux" is the resource ID
	d.Set("corge", resp.Corge)
	return nil
}
```

### Updating Resources

The `Update` property should be set to a
[`schema.UpdateFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#UpdateFunc),
which will be called whenever Terraform detects the resource has drifted from
the values defined in the HCL, but still exists in the state. The
`schema.UpdateFunc` receives a [`*schema.ResourceData`](resource-data.html)
and the `interface{}` returned by the provider's
[`ConfigureFunc`](new-provider.html#instantiating-clients). The function should
read the information it needs to update the resource from the
[`*schema.ResourceData`](resource-data.html) and make whatever API calls it
needs to to update the resource to match the `*schema.ResourceData`'s
description. If the resource is successfully updated, the function should
[update the `*schema.ResourceData`](resource-data.html#setting-state) with the
state as reported by the API and return `nil`. If an error is encountered, the
function should return an `error` describing the problem, which will be
surfaced to the user. The `*schema.ResourceData` should reflect the API's
output, to fill in any [`Computed` fields](schema.html#working-with-computed)
and detect any drift.

If the API does not support updating resources, only destroying and recreating
them, the `Update` property should be left unset, and all the fields of the
resource should be marked [`ForceNew`](schema.html#working-with-forcenew). 

If the API supports partial updates of resources (for example, using `PATCH`
requests) the `schema.UpdateFunc` should use the passed
`*schema.ResourceData`'s `HasChange` method to detect which properties need to
be updated, and `GetChange` to build the requests to make the necessary API
calls. If the API only allows resources to be replaced, not patched, use the
`*schema.ResourceData`'s `Get` and `GetOk` methods to build the requests as if
building a request for a `schema.CreateFunc`.

Terraform makes no effort to roll back failed updates, so `schema.UpdateFunc`s
that make multiple API calls or perform other multi-step transactional API
calls should make sure to use [partial state
mode](resource-data.html#partial-updates) on the `*schema.ResourceData`.

```go
func resourceBarCreate(d *schema.ResourceData, meta interface{}) error {
	// retrieve the information from the *schema.ResourceData
	quux := d.GetId().(string)

	// cast meta to the client we returned from the ConfigureFunc
	client := meta.(fooClient)

	if d.HasChange("corge") {
		// ignore the value in the state, we don't need it
		_, newCorge := d.GetChange("corge")
		req := client.UpdateBar(quux, newCorge)
		resp, err := req.Execute()
		if err != nil {
			// return an error
			return fmt.Sprintf("Error updating bar :%s", err.Error())
		}
		// update the *schema.ResourceData
		d.Set("corge", resp.Corge)
	}
	return nil
}
```

### Deleting Resources

The `Delete` property should be set to a
[`schema.DeleteFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#DeleteFunc),
which will be called whenever Terraform detects a resource in the statefile
that doesn't exist in the config file. The `schema.DeleteFunc` receives a
[`*schema.ResourceData`](resource-data.html) and the `interface{}` returned by
the provider's [`ConfigureFunc`](new-provider.html#instantiating-clients). The
function should read the information it needs to delete the resource from the
[`*schema.ResourceData`](resource-data.html) and make whatever API calls it
needs to to delete the resource. If the resource is successfully deleted, the
function should [set its ID to the empty string in the
`*schema.ResourceData`](resource-data.html#working-with-ids), which will remove
it from the statefile.

```go
func resourceBarDelete(d *schema.ResourceData, meta interface{}) error {
	// retrieve the information from the *schema.ResourceData
	quux := d.GetId().(string)

	// cast meta to the client we returned from the ConfigureFunc
	client := meta.(fooClient)
	
	// build our request -- this will look different for each provider
	req := client.DeleteBar(quux)

	// make our API request
	resp, err := req.Execute()
	if err != nil {
		// return an error
		return fmt.Sprintf("Error deleting bar: %s", err.Error())
	}
	
	// update the *schema.ResourceData
	d.SetId("") // remove the resource
	return nil
}
```

### Checking If Resources Exist

The `Exists` property should be set to a
[`schema.ExistsFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ExistsFunc),
which will be called whenever Terraform needs to check if a resource exists in
the API. The `schema.ExistsFunc` receives a
[`*schema.resourceData`](resource-data.html) and the `interface{}` returned by
the provider's [`ConfigureFunc`](new-provider.html#instantiating-clients). The
function should read the information it needs to check for the existence of the
resource from the [`*schema.ResourceData`](resource-data.html) and make
whatever API calls it needs to determine whether or not the resource exists. If
the function successfully determines the resource exists, it should return
`true`, with a `nil` `error`. If the function successfully determines the
resource does not exists, it should return `false`, with a `nil` `error`. If
the function encounters an error checking the existence of the resource, it
must return `true`, with an `error` describing the problem it ran into, which
will be surfaced to the user.

~> Any resource that has a `schema.ExistsFunc` that returns false will be
removed from the statefile, whether the `error` is `nil` or not.

~> **Note:** The `schema.ExistsFunc` must not modify the `*schema.ResourceData`
passed to it.

```go
func resourceBarExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// retrieve the information from the *schema.ResourceData
	quux := d.GetId().(string)

	// cast meta to the client we returned from the ConfigureFunc
	client := meta.(fooClient)
	
	// build our request -- this will look different for each provider
	req := client.GetBar(quux)

	// make our API request
	resp, err := req.Execute()
	if err != nil && err != foo.BarNotFound {
		// return an error
		return true, fmt.Sprintf("Error checking existence of bar: %s", err.Error())
	} else if err == foo.BarNotFound {
		return false, nil
	}
	
	return true, nil
}
```
