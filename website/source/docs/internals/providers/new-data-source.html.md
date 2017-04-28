---
layout: "docs"
page_title: "Creating a New Data Source"
sidebar_current: "docs-internals-provider-guide-new-data-source"
description: |-
  How to get started adding a data source to an existing provider.
---

# Creating Data Sources

Once we have a Provider, we can add some Data Sources to that provider, which
will help it pull information out of your infrastructure to be used in your
Resources.  To do this, you need to write some code in the package your
Provider is defined in.

In the built-in plugins for Terraform that are shipped as part of the codebase,
we have a file naming guideline: we use
`data_source_{api}_{data_source_name}.go` as the template to name our files,
and tend to stick to  single Data Source per file. For example, AWS has
`data_source_aws_acm_certificate.go`, because it reads certificates from AWS’
ACM service. Some providers only have a single API, so we use the provider name
in that case. For example, Docker has `data_source_docker_registry_image.go`.
If you want to contribute your code back to the Terraform repo, it makes things
easier if you follow this convention.

Inside this file, you’re going to define a function that takes no arguments and
returns a `*schema.Resource`, just like Resources.  While you can technically
name your function anything, we recommend (especially if you plan on
contributing your code back to the Terraform repo!) that you name it the
camelCase version of the file name. For example, `data_source_aws_eip.go` would
contain `dataSourceAwsEip`. We’ll be defining a _lot_ of functions in a
Provider, and this kind of naming scheme really helps keep things organised and
easy to find.

## Registering the Data Source  With the Provider

Now that you’ve got your function, switch back to your Provider definition, and
add the Data Source to the Provider’s `DataSourcesMap` property. The key will
be the name of the Data Source used in state and configuration files, and (by
convention) should match the `{data_source_name}` part used in the filename for
the file that defines the Resource. Nothing will necessarily break if you don’t
do this, but if you happen to accidentally have a collision with any other
resource in any other provider, things will likely break. This convention helps
keep things neat and orderly while avoiding conflicts.

## Defining the Data Source Properties

Each Data Source has a set of properties called the “schema” that is stored in
the state, just like Resources do. Think of it as the type definition for the
Data Source. For example, an AWS elastic IP data source has a `public_ip`
property, to allow configurations to access the IP, Docker’s registry image
data source has a `sha256_digest` property to access the checksum of the image,
and so on.

These properties are defined in the `Schema` property of the Data Source
definition. It takes a map with the keys being the property name and the values
being `*Schema` structs. The `*Schema` structs define some type information
(what kind of data to expect, etc.) along with some [advanced
behaviour](/docs/internals/providers/schema.html) for resources that helps
Terraform do the right thing without you needing to write a bunch of code.

~> **Note:** “id” is a reserved property name. Don’t call your property “id”.

## Calling the API

Now that we have a Data Source fully defined, it’s time to make it do
something. We’re going to use the Provider’s API client (possibly
[configured](/docs/internals/providers/new-provider.html#configuring-your-provider)
by the Provider’s `ConfigureFunc`) to read some resources. Though the Data
Source, as a `*Resource` type, has `Create`, `Read`, `Update`, `Delete`, and
`Exists` properties, only the function defined in the `Read` property will be
used. The function takes a `*ResourceData` struct and an `interface{}` as
arguments, and returns an `error`. 

The `*ResourceData` struct represents the state of the Resource as it should
be. It’s an amalgamation of several different sources of data, which are
explained further in [Understanding
ResourceData](/docs/internals/providers/resource-data.html). For Data Sources,
the `*ResourceData` struct is used to pull the identifying information
necessary to retrieve the information requested from the API, and as a place to
store the information the API returns.

The `interface{}` is the same `interface{}` returned by the Provider’s
`ConfigureFunc`. Typically, this is where you’d put the configured API client,
for example.

The `Read` function should then use the passed `*ResourceData` to retrieve the
ID (or other identifying information necessary for the API call) of the Data
Source to be read. It should then retrieve the Data Source from the API, and
set the `*ResourceData` to match.
