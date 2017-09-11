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

### Sample Request

```text
$ curl \
    https://registry.terraform.io/v1/modules/examplecorp/vapordb
```

### Sample Response

```json
[
  {
    "id": "examplecorp/vapordb/aws/1.0.0",
    "owner": "wispy",
    "namespace": "examplecorp",
    "name": "vapordb",
    "version": "1.0.0",
    "provider": "aws",
    "description": "Terraform Module for running VaporDB on AWS",
    "source": "https://github.com/examplecorp/terraform-aws-vapordb",
    "published_at": "2017-09-01T22:30:19.181077Z",
    "downloads": 2,
    "verified": true
  },
  {
    "id": "examplecorp/vapordb/azurerm/1.0.0",
    "owner": "wispy",
    "namespace": "examplecorp",
    "name": "vapordb",
    "version": "1.0.0",
    "provider": "azurerm",
    "description": "Terraform Module for running VaporDB on Azure",
    "source": "https://github.com/examplecorp/terraform-azurerm-vapordb",
    "published_at": "2017-09-01T22:30:19.181077Z",
    "downloads": 2,
    "verified": true
  }
]
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
    https://registry.terraform.io/v1/modules/Azure/network/azurerm
```

### Sample Response

```json
{
  "id": "Azure/network/azurerm/0.9.3",
  "owner": "echuvyrov",
  "namespace": "Azure",
  "name": "network",
  "version": "0.9.3",
  "provider": "azurerm",
  "description": "Terraform Azure RM Module for Network",
  "source": "https://github.com/Azure/terraform-azurerm-network",
  "published_at": "2017-09-01T22:30:19.181077Z",
  "downloads": 0,
  "verified": false,
  "root": {
    "path": "",
    "readme": "Create a basic network in Azure\n==============================================================================\n\nThis Terraform module deploys a Virtual Network in Azure with the following characteristics: ...",
    "empty": false,
    "inputs": [
      {
        "name": "resource_group_name",
        "description": "Default resource group name that the network will be created in.",
        "default": "\"myapp-rg\""
      },
      {
        "name": "location",
        "description": "The location/region where the core network will be created. The full list of Azure regions can be found at https://azure.microsoft.com/regions",
        "default": ""
      },
      {
        "name": "address_space",
        "description": "The address space that is used by the virtual network.",
        "default": "\"10.0.0.0/16\""
      },
      {
        "name": "dns_servers",
        "description": "The DNS servers to be used with vNet.",
        "default": "[]"
      },
      {
        "name": "subnet_prefixes",
        "description": "The address prefix to use for the subnet.",
        "default": "[\n  \"10.0.1.0/24\"\n]"
      },
      {
        "name": "subnet_names",
        "description": "A list of public subnets inside the vNet.",
        "default": "[\n  \"subnet1\"\n]"
      },
      {
        "name": "tags",
        "description": "The tags to associate with your network and subnets.",
        "default": "{\n  \"tag1\": \"\",\n  \"tag2\": \"\"\n}"
      },
      {
        "name": "allow_rdp_traffic",
        "description": "This optional variable, when set to true, adds a security rule allowing RDP traffic to flow through to the newly created network. The default value is false.",
        "default": "false"
      },
      {
        "name": "allow_ssh_traffic",
        "description": "This optional variable, when set to true, adds a security rule allowing SSH traffic to flow through to the newly created network. The default value is false.",
        "default": "false"
      }
    ],
    "outputs": [
      {
        "name": "vnet_id",
        "description": "The id of the newly created vNet"
      },
      {
        "name": "vnet_name",
        "description": "The Name of the newly created vNet"
      },
      {
        "name": "vnet_location",
        "description": "The location of the newly created vNet"
      },
      {
        "name": "vnet_address_space",
        "description": "The address space of the newly created vNet"
      },
      {
        "name": "vnet_dns_servers",
        "description": "The DNS servers of the newly created vNet"
      },
      {
        "name": "vnet_subnets",
        "description": "The ids of subnets created inside the newl vNet"
      },
      {
        "name": "security_group_id",
        "description": "The id of the security group attached to subnets inside the newly created vNet. Use this id to associate additional network security rules to subnets."
      }
    ],
    "dependencies": [],
    "resources": [
      {
        "name": "network",
        "type": "azurerm_resource_group"
      },
      {
        "name": "vnet",
        "type": "azurerm_virtual_network"
      },
      {
        "name": "subnet",
        "type": "azurerm_subnet"
      },
      {
        "name": "security_group",
        "type": "azurerm_network_security_group"
      },
      {
        "name": "security_rule_rdp",
        "type": "azurerm_network_security_rule"
      },
      {
        "name": "security_rule_ssh",
        "type": "azurerm_network_security_rule"
      }
    ]
  },
  "submodules": null,
  "providers": [
    "azurerm"
  ],
  "versions": [
    "0.9.2",
    "0.9.3"
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
    https://registry.terraform.io/v1/modules/Azure/network/azurerm/0.9.2
```

### Sample Response

```json
{
  "id": "Azure/network/azurerm/0.9.2",
  "owner": "echuvyrov",
  "namespace": "Azure",
  "name": "network",
  "version": "0.9.2",
  "provider": "azurerm",
  "description": "Terraform Azure RM Module for Network",
  "source": "https://github.com/Azure/terraform-azurerm-network",
  "published_at": "2017-08-30T22:22:12.222113Z",
  "downloads": 0,
  "verified": false,
  "root": {
    "path": "",
    "readme": "Create a basic network in Azure\n==============================================================================\n\nThis Terraform module deploys a Virtual Network in Azure with the following characteristics: ...",
    "empty": false,
    "inputs": [
      {
        "name": "tags",
        "description": "The tags to associate with your network and subnets.",
        "default": "{\n  \"tag1\": \"\",\n  \"tag2\": \"\"\n}"
      },
      {
        "name": "subnet_names",
        "description": "A list of public subnets inside the vNet.",
        "default": "[\n  \"subnet1\"\n]"
      },
      {
        "name": "subnet_prefixes",
        "description": "The address prefix to use for the subnet.",
        "default": "[\n  \"10.0.1.0/24\"\n]"
      },
      {
        "name": "dns_servers",
        "description": "The DNS servers to be used with vNet.",
        "default": "[]"
      },
      {
        "name": "address_space",
        "description": "The address space that is used by the virtual network.",
        "default": "\"10.0.0.0/16\""
      },
      {
        "name": "location",
        "description": "The location/region where the core network will be created. The full list of Azure regions can be found at https://azure.microsoft.com/regions",
        "default": ""
      },
      {
        "name": "prefix",
        "description": "Default prefix to use with your resource names.",
        "default": "\"myapp\""
      }
    ],
    "outputs": [
      {
        "name": "vnet_id",
        "description": "The id of the newly created vNet"
      },
      {
        "name": "vnet_name",
        "description": "The Name of the newly created vNet"
      },
      {
        "name": "vnet_location",
        "description": "The location of the newly created vNet"
      },
      {
        "name": "vnet_address_space",
        "description": "The address space of the newly created vNet"
      },
      {
        "name": "vnet_dns_servers",
        "description": "The DNS servers of the newly created vNet"
      },
      {
        "name": "vnet_subnets",
        "description": "The ids of subnets created inside the newl vNet"
      },
      {
        "name": "security_group_id",
        "description": "The id of the security group attached to subnets inside the newly created vNet. Use this id to associate additional network security rules to subnets."
      }
    ],
    "dependencies": [],
    "resources": [
      {
        "name": "security_rule_ssh",
        "type": "azurerm_network_security_rule"
      },
      {
        "name": "security_rule_rdp",
        "type": "azurerm_network_security_rule"
      },
      {
        "name": "security_group",
        "type": "azurerm_network_security_group"
      },
      {
        "name": "subnet",
        "type": "azurerm_subnet"
      },
      {
        "name": "vnet",
        "type": "azurerm_virtual_network"
      },
      {
        "name": "rg",
        "type": "azurerm_resource_group"
      }
    ]
  },
  "submodules": null,
  "providers": [
    "azurerm"
  ],
  "versions": [
    "0.9.2",
    "0.9.3"
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
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/1.0.0/download
```

### Sample Response

```text
HTTP/1.1 204 No Content
Content-Length: 0
X-Terraform-Get: https://api.github.com/repos/Azure/terraform-azurerm-network/tarball/v0.9.2//*?archive=tar.gz
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
$ curl \
    https://registry.terraform.io/v1/modules/hashicorp/consul/aws/download
```

### Sample Response

```text
HTTP/1.1 302 Found
Location: /v1/modules/Azure/network/azurerm/0.9.3/download
```

