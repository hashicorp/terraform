---
layout: "docs"
page_title: "Creating Data Sources"
sidebar_current: "docs-internals-provider-guide-new-data-source"
description: |-
  How to get started adding a data source to an existing provider.
---

# Creating Data Sources

Data sources are components of providers that pull information from outside of
Terraform's managed resources, to make it available in configuration.  They
should exist in the same package as the provider.

The built-in plugins for Terraform that are shipped as part of the codebase
have a file naming guideline: they use
`data_source_{api}_{data_source_name}.go` as the template to name files, and
tend to stick to  single data source per file. For example, AWS has
`data_source_aws_acm_certificate.go`, because it reads certificates from AWS'
ACM service. Some providers only have a single API, so they use the provider
name in that case. For example, Docker has
`data_source_docker_registry_image.go`. Following these conventions makes it
easier to contribute code back to the Terraform codebase.

Just like resources, data sources are functions that take no arguments and
return a `*schema.Resource`. While functions can technically be named anything,
the Terraform codebase uses the camelCase version of the file name as the
function name. For example, `data_source_aws_eip.go` would contain
`dataSourceAwsEip`. Providers have a lot of functions in them, and these naming
conventions help keep things organised and easy to find.

## Registering the Data Source with the Provider

Registering a data source with a provider consists of adding the
`*schema.Resource` for the data source to the `DataSourcesMap` of the
`*schema.Provider`. The key is the name of the data source as it will appear in
state and configuration files, and it should (by convention) match the
`{data_source_name}` portion of the filename.  This is not a technical
requirement, but if any other resource in the provider uses the same key,
Terraform will likely break. This convention helps to avoid that situation.

## Defining the Data Source Properties

Each data source has a set of properties called the "schema" that is stored in
the state. Think of it as the type definition for the data source. For example,
an AWS elastic IP data source has a `public_ip` property, to allow
configurations to access the IP, Docker's registry image data source has a
`sha256_digest` property to access the checksum of the image, and so on.

The `Schema` property of the `*schema.Resource` defines these properties.  It
takes a map with the property name as the key and `*schema.Schema` structs as
the values. The `*Schema` structs define some type information (what kind of
data to expect, etc.) along with some [advanced
behaviour](/docs/internals/providers/schema.html) for resources that helps
Terraform do the right thing without each resource and data source needing a
bunch of custom code.

~> **Note:** "id" is a reserved property name. Do not use "id" as a property
name.

## Calling the API

The data source obtains its values by using the provider's API client (possibly
[configured](/docs/internals/providers/new-provider.html#configuring-your-provider)
by the provider's `ConfigureFunc`) to retrieve some resources. Though the data
source, as a `*schema.Resource` type, has `Create`, `Read`, `Update`, `Delete`,
and `Exists` properties, Terraform only uses the function defined in the `Read`
property.  The function takes a `*schema.ResourceData` struct and an
`interface{}` as arguments, and returns an `error`. 

The `*schema.ResourceData` struct represents the state of the resource as it
should be. It's an amalgamation of several different sources of data, which are
explained further in [Using
ResourceData](/docs/internals/providers/resource-data.html). Data sources use
the `*schema.ResourceData` struct to pull the identifying information necessary
to retrieve the information requested from the API, and as a place to store the
information the API returns.

The `interface{}` is the same `interface{}` returned by the
`*schema.Provider`'s `ConfigureFunc`. It generally contains a configured API
client or similar form of access for the provider.

The `Read` function should use the passed `*ResourceData` to retrieve the ID
(or other identifying information necessary for the API call) of the data
source to be read. It should then retrieve the data source from the API, and
set the fields in `*schema.ResourceData` to match the response.
