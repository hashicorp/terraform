---
layout: "registry"
page_title: "Terraform Registry - HTTP API"
sidebar_current: "docs-registry-api"
description: |-
  The /acl endpoints create, update, destroy, and query ACL tokens in Consul.
---

# HTTP API

When downloading modules from registry sources such as the public
[Terraform Registry](https://registry.terraform.io), Terraform expects
the given hostname to support the following module registry protocol.

A registry module source is of the form `hostname/namespace/name/provider`,
where the initial hostname portion is implied to be `registry.terraform.io/`
if not specified. The public Terraform Registry is therefore the default
module source.

[Terraform Registry](https://registry.terraform.io) implements a superset
of this API to allow for importing new modules, etc, but any endpoints not
documented on this page are subject to change over time.

## Service Discovery

The hostname portion of a module source string is first passed to
[the service discovery protocol](/docs/internals/remote-service-discovery.html)
to determine if the given host has a module registry and, if so, the base
URL for its module registry endpoints.

The service identifier for this protocol is `modules.v1`, and the declared
URL should always end with a slash such that the paths shown in the following
sections can be appended to it.

For example, if discovery produces the URL `https://modules.example.com/v1/`
then this API would use full endpoint URLs like
`https://modules.example.com/v1/{namespace}/{name}/{provider}/versions`.

## Base URL

The example request URLs shown in this document are for the public [Terraform
Registry](https://registry.terraform.io), and use its API `<base_url>` of
`https://registry.terraform.io/v1/modules/`. Note that although the base URL in
the [discovery document](#service-discovery) _may include_ a trailing slash, we
include a slash after the placeholder in the `Path`s below for clarity.

## List Modules

These endpoints list modules according to some criteria.

| Method | Path                                  | Produces                   |
| ------ | ------------------------------------- | -------------------------- |
| `GET`  | `<base_url>`                          | `application/json`         |
| `GET`  | `<base_url>/:namespace`               | `application/json`         |

### Parameters

- `namespace` `(string: <optional>)` - Restricts listing to modules published by
  this user or organization. This is optionally specified as part of the URL
  path.

### Query Parameters

- `offset`, `limit` `(int: <optional>)` - See [Pagination](#pagination) for details.
- `provider` `(string: <optional>)` - Limits modules to a specific provider.
- `verified` `(bool: <optional>)` - If `true`, limits results to only verified
  modules. Any other value including none returns all modules _including_
  verified ones.

### Sample Request

```text
$ curl 'https://registry.terraform.io/v1/modules&limit=2&verified=true'
```

### Sample Response

```json
{
  "meta": {
    "limit": 2,
    "current_offset": 0,
    "next_offset": 2,
    "next_url": "/v1/modules?limit=2&offset=2&verified=true"
  },
  "modules": [
    {
      "id": "GoogleCloudPlatform/lb-http/google/1.0.4",
      "owner": "",
      "namespace": "GoogleCloudPlatform",
      "name": "lb-http",
      "version": "1.0.4",
      "provider": "google",
      "description": "Modular Global HTTP Load Balancer for GCE using forwarding rules.",
      "source": "https://github.com/GoogleCloudPlatform/terraform-google-lb-http",
      "published_at": "2017-10-17T01:22:17.792066Z",
      "downloads": 213,
      "verified": true
    },
    {
      "id": "terraform-aws-modules/vpc/aws/1.5.1",
      "owner": "",
      "namespace": "terraform-aws-modules",
      "name": "vpc",
      "version": "1.5.1",
      "provider": "aws",
      "description": "Terraform module which creates VPC resources on AWS",
      "source": "https://github.com/terraform-aws-modules/terraform-aws-vpc",
      "published_at": "2017-11-23T10:48:09.400166Z",
      "downloads": 29714,
      "verified": true
    }
  ]
}
```

## Search Modules

This endpoint allows searching modules.

| Method | Path                                  | Produces                   |
| ------ | ------------------------------------- | -------------------------- |
| `GET`  | `<base_url>/search`                   | `application/json`         |

### Query Parameters

- `q` `(string: <required>)` - The search string. Search syntax understood
  depends on registry implementation. The public registry supports basic keyword
  or phrase searches.
- `offset`, `limit` `(int: <optional>)` - See [Pagination](#pagination) for details.
- `provider` `(string: <optional>)` - Limits results to a specific provider.
- `namespace` `(string: <optional>)` - Limits results to a specific namespace.
- `verified` `(bool: <optional>)` - If `true`, limits results to only verified
  modules. Any other value including none returns all modules _including_
  verified ones.

### Sample Request

```text
$ curl 'https://registry.terraform.io/v1/modules/search?q=network&limit=2'
```

### Sample Response

```json
{
  "meta": {
    "limit": 2,
    "current_offset": 0,
    "next_offset": 2,
    "next_url": "/v1/modules/search?limit=2&offset=2&q=network"
  },
  "modules": [
    {
      "id": "zoitech/network/aws/0.0.3",
      "owner": "",
      "namespace": "zoitech",
      "name": "network",
      "version": "0.0.3",
      "provider": "aws",
      "description": "This module is intended to be used for configuring an AWS network.",
      "source": "https://github.com/zoitech/terraform-aws-network",
      "published_at": "2017-11-23T15:12:06.620059Z",
      "downloads": 39,
      "verified": false
    },
    {
      "id": "Azure/network/azurerm/1.1.1",
      "owner": "",
      "namespace": "Azure",
      "name": "network",
      "version": "1.1.1",
      "provider": "azurerm",
      "description": "Terraform Azure RM Module for Network",
      "source": "https://github.com/Azure/terraform-azurerm-network",
      "published_at": "2017-11-22T17:15:34.325436Z",
      "downloads": 1033,
      "verified": true
    }
  ]
}
```

## List Available Versions for a Specific Module

This is the primary endpoint for resolving module sources, returning the
available versions for a given fully-qualified module.

| Method | Path                                  | Produces                   |
| ------ | ------------------------------------- | -------------------------- |
| `GET`  | `<base_url>/:namespace/:name/:provider/versions` | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user or organization the module is
  owned by. This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

- `provider` `(string: <required>)` - The name of the provider.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl https://registry.terraform.io/v1/modules/hashicorp/consul/aws/versions
```

### Sample Response

The `modules` array in the response always includes the requested module as the
first element. Other elements of this list, if present, are dependencies of the
requested module that are provided to potentially avoid additional requests to
resolve these modules.

Additional modules are not required to be provided but, when present, can be
used by Terraform to optimize the module installation process.

Each returned module has an array of available versions, which Terraform
matches against any version constraints given in configuration.

```json
{
   "modules": [
      {
         "source": "hashicorp/consul/aws",
         "versions": [
            {
               "version": "0.0.1",
               "submodules" : [
                  {
                     "path": "modules/consul-cluster",
                     "providers": [
                        {
                           "name": "aws",
                           "version": ""
                        }
                     ],
                     "dependencies": []
                  },
                  {
                     "path": "modules/consul-security-group-rules",
                     "providers": [
                        {
                           "name": "aws",
                           "version": ""
                        }
                     ],
                     "dependencies": []
                  },
                  {
                     "providers": [
                        {
                           "name": "aws",
                           "version": ""
                        }
                     ],
                     "dependencies": [],
                     "path": "modules/consul-iam-policies"
                  }
               ],
               "root": {
                  "dependencies": [],
                  "providers": [
                     {
                        "name": "template",
                        "version": ""
                     },
                     {
                        "name": "aws",
                        "version": ""
                     }
                  ]
               }
            }
         ]
      }
   ]
}
```

## Download Source Code for a Specific Module Version

This endpoint downloads the specified version of a module for a single provider.

A successful response has no body, and includes the location from which the module
version's source can be downloaded in the `X-Terraform-Get` header. Note that
this string may contain special syntax interpreted by Terraform via
[`go-getter`](https://github.com/hashicorp/go-getter). See the [`go-getter`
documentation](https://github.com/hashicorp/go-getter#url-format) for details.

The value of `X-Terraform-Get` may instead be a relative URL, indicated by
beginning with `/`, `./` or `../`, in which case it is resolved relative to
the full URL of the download endpoint.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `<base_url>/:namespace/:name/:provider/:version/download` | `application/json`         |

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

## List Latest Version of Module for All Providers

This endpoint returns the latest version of each provider for a module.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `<base_url>/:namespace/:name`           | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user or organization the module is
  owned by. This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

### Query Parameters

- `offset`, `limit` `(int: <optional>)` - See [Pagination](#pagination) for details.

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

## Latest Version for a Specific Module Provider

This endpoint returns the latest version of a module for a single provider.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `<base_url>/:namespace/:name/:provider` | `application/json`         |

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
| `GET`  | `<base_url>/:namespace/:name/:provider/:version` | `application/json`         |

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

## Download the Latest Version of a Module

This endpoint downloads the latest version of a module for a single provider.

It returns a 302 redirect whose `Location` header redirects the client to the
download endpoint (above) for the latest version.

| Method | Path                         | Produces                   |
| ------ | ---------------------------- | -------------------------- |
| `GET`  | `<base_url>/:namespace/:name/:provider/download` | `application/json`         |

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