---
layout: "docs"
page_title: "Creating Data Sources"
sidebar_current: "docs-internals-provider-guide-new-data-source"
description: |-
  How to get started adding a data source to an existing provider.
---

# Creating Data Sources

Data sources are components of providers that pull information from external
sources, to make it available in configuration. They should exist in the same
package as the provider.

The built-in plugins for Terraform have a file naming guideline: they use
`data_source_<API>_<DATA_SOURCE_NAME>.go` as the template to name files, and
tend to stick to single data source per file. For example, AWS has
`data_source_aws_acm_certificate.go`, because it reads certificates from AWS'
ACM service. Some providers only have a single API, so they use the provider
name in that case. For example, Docker has
`data_source_docker_registry_image.go`. Following these conventions makes it
easier to contribute code back to the Terraform codebase.

Just like resources, data sources are functions that take no arguments and
return a `*schema.Resource`. While functions can be named anything,
the Terraform codebase uses the camelCase version of the file name as the
function name. For example, `data_source_aws_eip.go` would contain
`dataSourceAwsEip`. Providers have a lot of functions in them, and these naming
conventions help keep things organised and easy to find.

## Registering the Data Source with the Provider

Registering a data source with a provider consists of adding the
`*schema.Resource` for the data source to the `DataSourcesMap` of the
`*schema.Provider`. The key is the name of the data source as it will appear in
state and configuration files, and it should (by convention) match the
`<DATA_SOURCE_NAME>` portion of the filename. This is not a technical
requirement, but if any other resource in the provider uses the same key,
Terraform will likely break. This convention helps to avoid that situation.

## Defining the Data Source Properties

Each data source has a set of properties called the "schema" that is stored in
the state. Think of it as the type definition for the data source. For example,
an AWS elastic IP data source has a `public_ip` property, to allow
configurations to access the IP, Docker's registry image data source has a
`sha256_digest` property to access the checksum of the image, and so on.

The `Schema` property of the `*schema.Resource` defines these properties. It
takes a map with the property name as the key and `*schema.Schema` structs as
the values. The `*Schema` structs define some type information (what kind of
data to expect, etc.) along with some [advanced
behaviour](/docs/internals/providers/schema.html) for resources that helps
Terraform do the right thing without each resource and data source needing a
bunch of custom code.

~> **Note:** "id" is a reserved property name. Do not use "id" as a property
name.

## Calling the API

Data sources define their behavior through a [`schema.ReadFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ReadFunc) that will be called whenever Terraform needs to retrieve
values from the data source. Though the data source is implemented as a `*schema.Resource`,
the `Create`, `Update`, and `Delete` properties are all ignored, and should not be set.

The `Read` property should be set to a
[`schema.ReadFunc`](https://godoc.org/github.com/hashicorp/terraform/helper/schema#ReadFunc),
which receives a [`*schema.ResourceData`](resource-data.html) and the
`interface{}` returned from the provider's
[`ConfigureFunc`](new-provider.html#instantiating-clients). The function should
read the information it needs to retrieve the resource from the
[`*schema.ResourceData`](resource-data.html) and make whatever API calls it
needs to retrieve the resource described. If the resource is successfully
retrieved, the function should [update the
`*schema.ResourceData`](resource-data.html#setting-state) with the state as
reported by the API and return `nil`. If an error is encountered, the function
should return an `error` describing the problem, which will be surfaced to the
user.
