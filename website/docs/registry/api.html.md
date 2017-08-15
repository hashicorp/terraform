---
layout: "registry"
page_title: "Terraform Registry - HTTP API"
sidebar_current: "docs-registry-api"
description: |-
  The /acl endpoints create, update, destroy, and query ACL tokens in Consul.
---

# HTTP API

The [Terraform Registry](https://registry.terraform.io) has an HTTP API for
reading and downloading registry modules.

Terraform interacts with the registry only as read-only. Therefore, the
documented API is only read-only.
Any endpoints that aren't documented on this
page can and will likely change over time. This allows differing methods
for getting modules into a registry while keeping a consistent API for
reading modules in a registry.

## List Latest Version of Module for All Providers

This endpoint returns the latest version of each provider for a module.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `/v1/modules/:namespace/:name` | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user the module is owned by.
  This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul
```

### Sample Response

```json
TODO
```

## Latest Module for a Single Provider

This endpoint returns the latest version of a module for a single provider.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `/v1/modules/:namespace/:name/:provider` | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user the module is owned by.
  This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

- `provider` `(string: <required>)` - The name of the provider.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws
```

### Sample Response

```json
TODO
```

## Get a Specific Module

This endpoint returns the specified version of a module for a single provider.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `/v1/modules/:namespace/:name/:provider/:version` | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user the module is owned by.
  This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

- `provider` `(string: <required>)` - The name of the provider.
  This is required and is specified as part of the URL path.

- `version` `(string: <required>)` - The version of the module.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/1.0.0
```

### Sample Response

```json
TODO
```

## Download a Specific Module

This endpoint downloads the specified version of a module for a single provider.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `/v1/modules/:namespace/:name/:provider/:version/download` | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user the module is owned by.
  This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

- `provider` `(string: <required>)` - The name of the provider.
  This is required and is specified as part of the URL path.

- `version` `(string: <required>)` - The version of the module.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/1.0.0/download
```

### Sample Response

```json
TODO
```

## Download the Latest Version of a Module

This endpoint downloads the latest version of a module for a single provider.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `/v1/modules/:namespace/:name/:provider/download` | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user the module is owned by.
  This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

- `provider` `(string: <required>)` - The name of the provider.
  This is required and is specified as part of the URL path.

- `version` `(string: <required>)` - The version of the module.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/download
```

### Sample Response

```json
TODO
```

