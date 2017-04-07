---
layout: "enterprise"
page_title: "Configurations - API - Terraform Enterprise"
sidebar_current: "docs-enterprise-api-configurations"
description: |-
  A configuration represents settings associated with a resource that runs
  Terraform with versions of Terraform configuration.
---

# Configuration API

A configuration version represents versions of Terraform configuration. Each set
of changes to Terraform HCL files or the scripts used in the files should have
an associated configuration version.

When creating versions via the API, the variables attribute can be sent to
include the necessary variables for the Terraform configuration. A configuration
represents settings associated with a resource that runs Terraform with versions
of Terraform configuration. Configurations have many configuration versions
which represent versions of Terraform configuration templates and other
associated configuration. Most operations take place on the configuration
version, not the configuration.

## Get Latest Configuration Version

This endpoint gets the latest configuration version.

| Method | Path           |
| :----- | :------------- |
| `GET`  | `/terraform/configurations/:username/:name/versions/latest` |

### Parameters

- `:username` `(string: <required>)` - Specifies the username or organization
  name under which to get the latest configuration version. This username must
  already exist in the system, and the user must have permission to create new
  configuration versions under this namespace. This is specified as part of the
  URL.

- `:name` `(string: <required>)` - Specifies the name of the configuration for
  which to get the latest configuration. This is specified as part of the URL.

### Sample Request

```text
$ curl \
    --header "X-Atlas-Token: ..." \
    https://atlas.hashicorp.com/api/v1/terraform/configurations/my-organization/my-configuration/versions/latest
```

### Sample Response

```json
{
  "version": {
    "version": 6,
    "metadata": {
      "foo": "bar"
    },
    "tf_vars": [],
    "variables": {}
  }
}
```

- `version` `(int)` - the unique version instance number.

- `metadata` `(map<string|string>)` - a map of arbitrary metadata for this
  version.

## Create Configuration Version

This endpoint creates a new configuration version.

| Method | Path           |
| :----- | :------------- |
| `POST` | `/terraform/configurations/:username/:name/versions` |

### Parameters

- `:username` `(string: <required>)` - Specifies the username or organization
  name under which to create this configuration version. This username must
  already exist in the system, and the user must have permission to create new
  configuration versions under this namespace. This is specified as part of the
  URL.

- `:name` `(string: <required>)` - Specifies the name of the configuration for
  which to create a new version. This is specified as part of the URL.

- `metadata` `(map<string|string>)` - Specifies an arbitrary hash of key-value
  metadata pairs. This is specified as the payload as JSON.

- `variables` `(map<string|string>)` - Specifies a hash of key-value pairs that
  will be made available as variables to this version.

### Sample Payload

```json
{
  "version": {
    "metadata": {
      "git_branch": "master",
      "remote_type": "atlas",
      "remote_slug": "hashicorp/atlas"
    },
    "variables": {
      "ami_id": "ami-123456",
      "target_region": "us-east-1",
      "consul_count": "5",
      "consul_ami": "ami-123456"
    }
  }
}
```

### Sample Request

```text
$ curl \
    --request POST \
    --header "X-Atlas-Token: ..." \
    --header "Content-Type: application/json" \
    --data @payload.json \
    https://atlas.hashicorp.com/api/v1/terraform/configurations/my-organization/my-configuration/versions
```

### Sample Response

```json
{
  "version": 6,
  "upload_path": "https://binstore.hashicorp.com/ddbd7db6-f96c-4633-beb6-22fe2d74eeed",
  "token": "ddbd7db6-f96c-4633-beb6-22fe2d74eeed"
}
```

- `version` `(int)` - the unique version instance number. This is
  auto-incrementing.

- `upload_path` `(string)` - the path where the archive should be uploaded via a
  `POST` request.

- `token` `(string)` - the token that should be used when uploading the archive
  to the `upload_path`.

## Check Upload Progress

This endpoint retrieves the progress for an upload of a configuration version.

| Method | Path           |
| :----- | :------------- |
| `GET` | `/terraform/configurations/:username/:name/versions/progress/:token` |

### Parameters

- `:username` `(string: <required>)` - Specifies the username or organization to
  read progress. This is specified as part of the URL.

- `:name` `(string: <required>)` - Specifies the name of the configuration for
  to read progress. This is specified as part of the URL.

- `:token` `(string: <required>)` - Specifies the token that was returned from
  the create option. **This is not an Atlas Token!** This is specified as part
  of the URL.

### Sample Request

```text
$ curl \
    --header "X-Atlas-Token: ..." \
    https://atlas.hashicorp.com/api/v1/terraform/configurations/my-organization/my-configuration/versions/progress/ddbd7db6-f96c-4633-beb6-22fe2d74eeed
```

### Sample Response
