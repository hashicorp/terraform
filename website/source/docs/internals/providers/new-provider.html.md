---
layout: "docs"
page_title: "Creating Providers"
sidebar_current: "docs-internals-provider-guide-new-provider"
description: |-
  How to get started building a new provider.
---

# Creating Providers

The
[`schema.Provider`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Provider)
type is the backbone of Terraform providers. Though it has many methods,
providers almost never call those methods, so they can be safely ignored.

## Creating the Package

Providers exist within their own packages. Inside the Terraform codebase, the
`builtin/providers` directory holds these packages, and the packages are named
for the provider they contain. Providers that do not ship with Terraform live
within a repository. The repository, by convention, should be named
`github.com/<YOUR_USERNAME>/terraform-provider-<NAME>`, though this is not a
hard requirement.

The new package, wherever it lives, holds a `provider.go` file containing a
`Provider` function. The `Provider` function takes no arguments and returns a
[`terraform.ResourceProvider`](https://godoc.org/github.com/hashicorp/terraform/terraform#ResourceProvider):

```go
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		// provider definition here
	}
}
```

The
[`*schema.Provider`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Provider)
type fills the
[`terraform.ResourceProvider`](https://godoc.org/github.com/hashicorp/terraform/terraform#ResourceProvider)
interface. It's not required that providers use this implementation, but it's
highly recommended.

## Configuring the Provider

### User-Supplied Information

Most API providers take some form of configuration: an API endpoint to use,
credentials to use, or other user-provided information. Terraform providers can
accept this information using the `Schema` property of the `*schema.Provider`
that defines the provider.

The key of the `Schema` property is the field name being described. For
example, for the following provider configuration:

```hcl
provider "foo" {
  username = "terraform"
}
```

The key of the `Schema` property should be `username`.

The field of the `Schema` property is a
[`*schema.Schema`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Schema)
representing the value expected. The above example could be represented like
this:

```go
provider := &schema.Provider{
	Schema: map[string]*Schema{
		"username": {
			Type: schema.TypeString,
			Required: true,
		},
	},
}
```

[Understanding the Schema](schema.html) has more information on the
`*schema.Schema` type.

### Instantiating Clients

Most API providers will also require some sort of instantiated client to
interact with them. It is generally desirable to instantiate these clients
once, both to avoid expensive setup costs and to share things like network
connections across the entire lifecycle.

Terraform providers have a `ConfigureFunc` property that can be set to a
[`schema.ConfigureFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ConfigureFunc).
This function accepts a [`*schema.ResourceData`](resource-data.html) and
returns an `interface{}` or an `error`. The function is called at the beginning
of the Terraform lifecycle. If it returns an error, Terraform will surface
that error to the user and exit. Otherwise, the returned `interface{}` will be
provided to the Create, Read, Update, Destroy, and Exists functions in the
provider.

The passed `*schema.ResourceData` contains the user-supplied values for the
`Schema` property of the `*schema.Provider`.

~> **Note:** Return errors from `ConfigureFunc` instead of using `panic` or
`os.Exit`.

## Registering a Provider

Terraform uses a [plugin architecture](/docs/plugins), meaning providers must
be registered before they are usable. Even providers that ship with Terraform
must be registered.

Providers that live within the Terraform codebase have a folder in the
`builtin/bins` directory named `provider-<NAME>`. Providers that live within
their own repositories can put their registration code anywhere, but convention
is to put the code in the root of the repository. The only requirement is that
the package containing the code is named `main`.

Registration for a provider happens in the `main` function, which is
responsible for serving the plugin. Any of the [built-in
plugins](https://github.com/hashicorp/terraform/tree/master/builtin/bins) in
Terraform can provide an example to follow. The function must import the
package the function that returns a `*schema.Provider` is in, and call
[`plugin.Serve`](https://godoc.org/github.com/hashicorp/terraform/plugin#Serve).
The
[`*plugin.ServeOpts`](https://godoc.org/github.com/hashicorp/terraform/plugin#ServeOpts)
passed to `plugin.Serve` should have the `ProviderFunc` property set to the
function that returns a `*schema.Provider`:

```go
package main

import (
	"github.com/hashicorp/terraform/builtin/providers/archive"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: archive.Provider,
	})
}
```
