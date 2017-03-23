---
layout: "api"
page_title: "API Documentation"
sidebar_current: "docs-enterprise-api"
description: |-
  Terraform Enterprise provides an API for a **subset of features**.
---

# Terraform Enterprise API Documentation

Terraform Enterprise provides an API for a **subset of features** available. For questions
or requests for new API features please email [support@hashicorp.com](mailto:support@hashicorp.com).

## Available Endpoints Terraform Enterprise

- [Environments](/docs/enterprise/api/environments.html)
- [Runs](/docs/enterprise/api/runs.html)
- [States](/docs/enterprise/api/states.html)

## Authentication

All requests must be authenticated with an `X-Atlas-Token` HTTP header. This 
token can be generated or revoked on the account tokens page. 
Your token will have access to all resources your account has access to.

For organization level resources, we recommend creating a separate user account
that can be added to the organization with the specific privilege level
required.

## Response Codes

Standard HTTP response codes are returned. `404 Not Found`
codes are returned for all resources that a user does not have access to,
as well as for resources that don't exist. This is done to avoid a
potential attacker discovering the existence of a resource.

## Errors

Errors are returned in JSON format:

    {
      "errors": {
        "name": [
          "has already been taken"
        ]
      }
    }

## Versioning

The API currently resides under the `/v1` prefix. Future APIs
will increment this version leaving the `/v1` API intact, though
in the future certain features may be deprecated. In that case,
ample notice to migrate to the new API will be provided.

## Content Type

The API accepts namespaced attributes in either
JSON or `application/x-www-form-urlencoded`. We recommend
using JSON, but for simplicity form style requests are supported.

Below is an equivalent example with both styles using `curl`.

### JSON Request Example

    $ cat variables.json
    {
      "variables": {
          "desired_capacity": "15",
          "foo": "bar"
      }
    }
    $ curl %{ATLAS_URL}/api/v1/environments/%{DEFAULT_USERNAME}/test/variables \
        -X PUT \
        -H 'Content-Type: application/json' \
        -d @variables.json \
        -H "X-Atlas-Token: $ATLAS_TOKEN"

### Form URL Encoded Example

    $ curl %{ATLAS_URL}/api/v1/environments/%{DEFAULT_USERNAME}/test/variables \
        -X PUT \
        -d variables[foo]='bar' \
        -d variables[desired_capacity]='15' \
        -H "X-Atlas-Token: $ATLAS_TOKEN"
