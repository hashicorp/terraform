---
layout: "enterprise"
page_title: "Runs - API - Terraform Enterprise"
sidebar_current: "docs-enterprise-api-runs"
description: |-
  Runs in Terraform Enterprise represents a two step Terraform plan and a subsequent apply.
---

# Runs API

Runs in Terraform Enterprise represents a two step Terraform plan and a
subsequent apply.

Runs are queued under [environments](/docs/enterprise/api/environments.html)
and require a two-step confirmation workflow. However, environments
can be configured to auto-apply to avoid this.

## Queue Run

Starts a new run (plan) in the environment. Requires a configuration version to
be present on the environment to succeed, but will otherwise 404.

| Method | Path           |
| :----- | :------------- |
| `POST` | `/environments/:username/:name/plan` |

### Parameters

- `:username` `(string: <required>)` - Specifies the username or organization
  name under which to get the latest configuration version. This username must
  already exist in the system, and the user must have permission to create new
  configuration versions under this namespace. This is specified as part of the
  URL.

- `:name` `(string: <required>)` - Specifies the name of the configuration for
  which to get the latest configuration. This is specified as part of the URL.

- `destroy` `(bool: false)` - Specifies if the plan should be a destroy plan.

### Sample Payload

```json
{
  "destroy": false
}
```

### Sample Request

```text
$ curl \
    --request POST \
    --header "X-Atlas-Token: ..." \
    --header "Content-Type: application/json" \
    --data @payload.json \
    https://atlas.hashicorp.com/api/v1/environments/my-organization/my-environment/plan
```

### Sample Response

```json
{
  "success": true
}
```
