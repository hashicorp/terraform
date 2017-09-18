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

Terraform interacts with the registry as read-only. Therefore, the documented
API is read-only. Any endpoints that aren't documented on this page can and will
likely change over time. This allows differing methods for getting modules into
the registry while keeping a consistent API for reading modules in the registry.

## List Latest Version of Module for All Providers

This endpoint returns the latest version of each provider for a module.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `/v1/modules/:namespace/:name` | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user or organization the module is
  owned by. This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

### Query Parameters

- `offset`, `limit` `(int: <optional>)` - See [Pagination](#Pagination) for details.

### Sample Request

```text
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul
```

### Sample Response

```json
{
  "meta": {
    "limit": 15,
    "current_offset": 0
  },
  "modules": [
    {
      "id": "hashicorp/consul/azurerm/0.0.1",
      "owner": "gruntwork-team",
      "namespace": "hashicorp",
      "name": "consul",
      "version": "0.0.1",
      "provider": "azurerm",
      "description": "A Terraform Module for how to run Consul on AzureRM using Terraform and Packer",
      "source": "https://github.com/hashicorp/terraform-azurerm-consul",
      "published_at": "2017-09-14T23:22:59.923047Z",
      "downloads": 100,
      "verified": false
    },
    {
      "id": "hashicorp/consul/aws/0.0.1",
      "owner": "gruntwork-team",
      "namespace": "hashicorp",
      "name": "consul",
      "version": "0.0.1",
      "provider": "aws",
      "description": "A Terraform Module for how to run Consul on AWS using Terraform and Packer",
      "source": "https://github.com/hashicorp/terraform-aws-consul",
      "published_at": "2017-09-14T23:22:44.793647Z",
      "downloads": 113,
      "verified": false
    }
  ]
}
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

Note this response has has some fields trimmed for clarity.

```json
{
  "id": "hashicorp/consul/aws/0.0.1",
  "owner": "gruntwork-team",
  "namespace": "hashicorp",
  "name": "consul",
  "version": "0.0.1",
  "provider": "aws",
  "description": "A Terraform Module for how to run Consul on AWS using Terraform and Packer",
  "source": "https://github.com/hashicorp/terraform-aws-consul",
  "published_at": "2017-09-14T23:22:44.793647Z",
  "downloads": 113,
  "verified": false,
  "root": {
    "path": "",
    "readme": "# Consul AWS Module\n\nThis repo contains a Module for how to deploy a [Consul]...",
    "empty": false,
    "inputs": [
      {
        "name": "ami_id",
        "description": "The ID of the AMI to run in the cluster. ...",
        "default": "\"\""
      },
      {
        "name": "aws_region",
        "description": "The AWS region to deploy into (e.g. us-east-1).",
        "default": "\"us-east-1\""
      }
    ],
    "outputs": [
      {
        "name": "num_servers",
        "description": ""
      },
      {
        "name": "asg_name_servers",
        "description": ""
      }
    ],
    "dependencies": [],
    "resources": []
  },
  "submodules": [
    {
      "path": "modules/consul-cluster",
      "readme": "# Consul Cluster\n\nThis folder contains a [Terraform](https://www.terraform.io/) ...",
      "empty": false,
      "inputs": [
        {
          "name": "cluster_name",
          "description": "The name of the Consul cluster (e.g. consul-stage). This variable is used to namespace all resources created by this module.",
          "default": ""
        },
        {
          "name": "ami_id",
          "description": "The ID of the AMI to run in this cluster. Should be an AMI that had Consul installed and configured by the install-consul module.",
          "default": ""
        }
      ],
      "outputs": [
        {
          "name": "asg_name",
          "description": ""
        },
        {
          "name": "cluster_size",
          "description": ""
        }
      ],
      "dependencies": [],
      "resources": [
        {
          "name": "autoscaling_group",
          "type": "aws_autoscaling_group"
        },
        {
          "name": "launch_configuration",
          "type": "aws_launch_configuration"
        }
      ]
    }
  ],
  "providers": [
    "aws",
    "azurerm"
  ],
  "versions": [
    "0.0.1"
  ]
}
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
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/0.0.1
```

### Sample Response

Note this response has has some fields trimmed for clarity.


```json
{
  "id": "hashicorp/consul/aws/0.0.1",
  "owner": "gruntwork-team",
  "namespace": "hashicorp",
  "name": "consul",
  "version": "0.0.1",
  "provider": "aws",
  "description": "A Terraform Module for how to run Consul on AWS using Terraform and Packer",
  "source": "https://github.com/hashicorp/terraform-aws-consul",
  "published_at": "2017-09-14T23:22:44.793647Z",
  "downloads": 113,
  "verified": false,
  "root": {
    "path": "",
    "readme": "# Consul AWS Module\n\nThis repo contains a Module for how to deploy a [Consul]...",
    "empty": false,
    "inputs": [
      {
        "name": "ami_id",
        "description": "The ID of the AMI to run in the cluster. ...",
        "default": "\"\""
      },
      {
        "name": "aws_region",
        "description": "The AWS region to deploy into (e.g. us-east-1).",
        "default": "\"us-east-1\""
      }
    ],
    "outputs": [
      {
        "name": "num_servers",
        "description": ""
      },
      {
        "name": "asg_name_servers",
        "description": ""
      }
    ],
    "dependencies": [],
    "resources": []
  },
  "submodules": [
    {
      "path": "modules/consul-cluster",
      "readme": "# Consul Cluster\n\nThis folder contains a [Terraform](https://www.terraform.io/) ...",
      "empty": false,
      "inputs": [
        {
          "name": "cluster_name",
          "description": "The name of the Consul cluster (e.g. consul-stage). This variable is used to namespace all resources created by this module.",
          "default": ""
        },
        {
          "name": "ami_id",
          "description": "The ID of the AMI to run in this cluster. Should be an AMI that had Consul installed and configured by the install-consul module.",
          "default": ""
        }
      ],
      "outputs": [
        {
          "name": "asg_name",
          "description": ""
        },
        {
          "name": "cluster_size",
          "description": ""
        }
      ],
      "dependencies": [],
      "resources": [
        {
          "name": "autoscaling_group",
          "type": "aws_autoscaling_group"
        },
        {
          "name": "launch_configuration",
          "type": "aws_launch_configuration"
        }
      ]
    }
  ],
  "providers": [
    "aws",
    "azurerm"
  ],
  "versions": [
    "0.0.1"
  ]
}
```

## Download a Specific Module

This endpoint downloads the specified version of a module for a single provider.

A successful response has no body, and includes the URL from which the module
version's source can be downloaded in the `X-Terraform-Get` header. Note that
this URL may contain special syntax interpreted by Terraform via
[`go-getter`](https://github.com/hashicorp/go-getter). See the [`go-getter`
documentation](https://github.com/hashicorp/go-getter#url-format) for details.

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
$ curl -i \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/0.0.1/download
```

### Sample Response

```text
HTTP/1.1 204 No Content
Content-Length: 0
X-Terraform-Get: https://api.github.com/repos/hashicorp/terraform-aws-consul/tarball/v0.0.1//*?archive=tar.gz
```

## Download the Latest Version of a Module

This endpoint downloads the latest version of a module for a single provider.

It returns a 302 redirect whose `Location` header redirects the client to the
download endpoint (above) for the latest version.

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
$ curl -i \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/download
```

### Sample Response

```text
HTTP/1.1 302 Found
Location: /v1/modules/hashicorp/consul/aws/0.0.1/download
Content-Length: 70
Content-Type: text/html; charset=utf-8

<a href="/v1/modules/hashicorp/consul/aws/0.0.1/download">Found</a>.
```

## HTTP Status Codes

The API follows regular HTTP status semantics. To make implementing a complete
client easier, some details on our policy and potential future status codes are
listed below. A robust client should consider how to handle all of the
following.

 - **Success:** Return status is `200` on success with a body or `204` if there
   is no body data to return.
 - **Redirects:** Moved or aliased endpoints redirect with a `301`. Endpoints
   redirecting to the latest version of a module may redirect with `302` or
   `307` to indicate that they logically point to different resources over time.
 - **Client Errors:** Invalid requests will receive the relevant `4xx` status.
   Except where noted below, the request should not be retried.
 - **Rate Limiting:** Clients placing excessive load on the service might be
   rate-limited and receive a `429` code. This should be interpreted as a sign
   to slow down, and wait some time before retrying the request.
 - **Service Errors:** The usual `5xx` errors will be returned for service
   failures. In all cases it is safe to retry the request after receiving a
   `5xx` response.
 - **Load Shedding:** A `503` response indicates that the service is under load
   and can't process your request immediately. As with other `5xx` errors you
   may retry after some delay, although clients should consider being more
   lenient with retry schedule in this case.

## Error Responses

When a `4xx` or `5xx` status code is returned. The response payload will look
like the following example:

```json
{
  "errors": [
    "something bad happened"
  ]
}
```

The `errors` key is a list containing one or more strings where each string
describes an error that has occurred.

Note that it is possible that some `5xx` errors might result in a response that
is not in JSON format above due to being returned by an intermediate proxy.

## Pagination

Endpoints that return lists of results use a common pagination format.

They accept positive integer query variables `offset` and `limit` which have the
usual SQL-like semantics. Each endpoint will have a sane default limit and a
default offset of `0`. Each endpoint will also apply a sane maximum limit,
requesting more results will just result in the maximum limit being used.

The response for a paginated result set will look like:

```json
{
  "meta": {
    "limit": 15,
    "current_offset": 15,
    "next_offset": 30,
    "prev_offset": 0,
  },
  "<object name>": []
}
```
Note that:
  - `next_offset` will only be present if there are more results available.
  - `prev_offset` will only be present if not at `offset = 0`.
  - `limit` is the actual limit that was applied, it may be lower than the requested limit param.
  - The key for the result array varies based on the endpoint and will be the
    type of result pluralized, for example `modules`.