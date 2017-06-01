---
layout: "enterprise"
page_title: "Environments - API - Terraform Enterprise"
sidebar_current: "docs-enterprise-api-environments"
description: |-
  Environments represent running infrastructure managed by Terraform.
---

# Environments API

Environments represent running infrastructure managed by Terraform.

Environments can also be connected to Consul clusters. This documentation covers
the environment interactions with Terraform.

## Get Latest Configuration Version

This endpoint updates the Terraform variables for an environment. Due to the
sensitive nature of variables, they are not returned on success.

| Method | Path           |
| :----- | :------------- |
| `PUT`  | `/environments/:username/:name/variables` |

### Parameters

- `:username` `(string: <required>)` - Specifies the username or organization
  name under which to update variables. This username must already exist in the
  system, and the user must have permission to create new configuration versions
  under this namespace. This is specified as part of the URL.

- `:name` `(string: <required>)` - Specifies the name of the environment for
  which to update variables. This is specified as part of the URL.

- `variables` `(map<string|string>)` - Specifies a key-value map of Terraform
  variables to be updated. Existing variables will only be removed when their
  value is empty. Variables of the same key will be overwritten.

    -> Note: Only string variables can be updated via the API currently. Creating or updating HCL variables is not yet supported.

### Sample Payload

```json
{
  "variables": {
    "desired_capacity": "15",
    "foo": "bar"
  }
}
```

### Sample Request

```text
$ curl \
    --header "X-Atlas-Token: ..." \
    --header "Content-Type: application/json" \
    --request PUT \
    --data @payload.json \
    https://atlas.hashicorp.com/api/v1/environments/my-organization/my-environment/variables
```

### Sample Response


```text
```

(empty body)
