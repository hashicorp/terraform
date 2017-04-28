---
layout: "docs"
page_title: "Creating Providers"
sidebar_current: "docs-internals-provider-guide-new-provider"
description: |-
  How to get started building a new provider.
---

# Creating Providers

Say you’ve found a new API provider that Terraform doesn’t support, and you
want to go ahead and implement it. First of all, thank you! It’s always
exciting when the community helps Terraform learn new tricks, and the Terraform
team appreciates all the contributions the community has made.

To get started, you’re going to want to look at the
[`Provider`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#Provider)
type. Don’t worry if it looks complex or confusing; we’re going to break it
down and walk through it over the course of this guide. There are a lot of
methods, but the good news is it’s not your job to call or define _any_ of
them. You can safely ignore them.

## Creating the Package

To define a new provider, you need to create a new package. If you are
contributing this to the Terraform codebase, create a directory named after
your plugin in the `builtin/providers` directory of Terraform. If you’re
planning on maintaining it outside of Terraform as a standalone plugin, create
a new repository for it&emdash;we recommend
`github.com/{yourname}/terraform-provider-{name}`, as that will simplify things
down the road.

In your new package, create a `provider.go` file. In that file, you’re going to
create a `Provider` function that takes no arguments and returns a
`terraform.ResourceProvider`:

```go
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		// your provider definition here
	}
}
```

The `*schema.Provider` you’re returning fills the `terraform.ResourceProvider`
interface; you don’t have to use it, you could use your own custom
implementation of the interface, but that’s tricky and hard to get right, so we
highly recommend you use the built-in `*schema.Provider`.

For the moment, just set the properties of the returned `*schema.Provider` to
empty maps (leave `ConfigureFunc` as nil). We’ll take care of the properties
later; let’s get that `ConfigureFunc` defined.

## Configuring Your Provider

Most API providers take some form of configuration: an API endpoint to use,
credentials to use, and so on. To accomplish this, we’re going to set up the
`ConfigureFunc` property on your Provider. If an API client needs to be
instantiated before it’s used in your Resources, this is the place to do it.

The premise is pretty simple: you’re going to define a function that takes a
`ResourceData` struct (more on that
[later](/docs/internals/providers/resource-data.html)) and returns an
`interface{}` or an `error`. If you don’t return an error, the interface you
return will be stored and passed to all your resources as a meta parameter. If
you do return an error, that will bubble up to the Terraform CLI and abort
whatever command is being run. **Note:** please return errors instead of using
`panic` or `os.Exit`.

## Registering Your Provider

Because Terraform uses a plugin architecture, we need to register our new
plugin with Terraform before Terraform knows how to use it. (Fun fact: even the
providers built into the codebase need to be registered!)

If you’re contributing your provider to the Terraform codebase, you need to
create a new folder in the `builtin/bins` directory in the Terraform codebase
named `provider-{name}`. Inside, you’ll put your `main.go` file. If you’re
maintaining your own plugin, you can put the `main.go` file in the root of the
repo containing the code for your provider, or really anywhere you want, as
long as it’s in package main.

Inside that `main.go` file, you’re going to define a `main` function that
serves your plugin. This is pretty straightforward, so check out any of the
[built-in
plugins](https://github.com/hashicorp/terraform/tree/master/builtin/bins) for
an example to follow. The only thing you should need to customise is the import
path to the package containing your `Provider` function and the `ProviderFunc`
property to point to that `Provider` function.
