---
layout: "enterprise"
page_title: "State - API - Terraform Enterprise"
sidebar_current: "docs-enterprise-api-states"
description: |-
  State represents the status of your infrastructure at the last time Terraform was run.
---

# State API

State represents the status of your infrastructure at the last time Terraform
was run. States can be pushed to Terraform Enterprise from Terraform's CLI after
an apply is done locally, or state is automatically stored if the apply is done
in Terraform Enterprise.

## List of States

This endpoint gets a list of states accessible to the user corresponding to the
provided token.

| Method | Path           |
| :----- | :------------- |
| `GET`  | `/terraform/state` |

### Parameters

- `?username` `(string: "")` - Specifies the organization/username to filter
  states

- `?page` `(int: 1)` - Specifies the pagination, which defaults to page 1.

### Sample Requests

```text
$ curl \
    --header "X-Atlas-Token: ..." \
    https://atlas.hashicorp.com/api/v1/terraform/state
```

```text
$ curl \
    --header "X-Atlas-Token: ..." \
    https://atlas.hashicorp.com/api/v1/terraform/state?username=acme
```

### Sample Response

```json
{
  "states": [
    {
      "updated_at": "2017-02-03T19:52:37.693Z",
      "environment": {
        "username": "my-organization",
        "name": "docs-demo-one"
      }
    },
    {
      "updated_at": "2017-04-06T15:48:49.677Z",
      "environment": {
        "username": "my-organization",
        "name": "docs-demo-two"
      }
    }
  ]
}
```
