---
layout: "api"
page_title: "Configuration Versions API"
sidebar_current: "docs-enterprise-api-configversions"
description: |-
  A configuration version represents versions of Terrraform configuration.
---

# Configuration Versions API

A configuration version represents versions of Terrraform configuration.
Each set of changes to Terraform HCL files or the scripts
used in the files should have an associated configuration version.

When creating versions via the API, the variables attribute can be sent
to include the necessary variables for the Terraform configuration.

### Configuration Version Attributes

<table class="apidocs">
  <tr>
    <th>Attribute</th>
    <th>Description</th>
    <th>Required</th>
  </tr>
  <tr>
    <td><code>variables</code></td>
    <td>A key/value map of Terraform variables to be associated
      with the configuration version.</td>
    <td>No</td>
  </tr>
  <tr>
    <td><code>metadata</code></td>
    <td>A hash of key value metadata pairs.</td>
    <td>No</td>
  </tr>
</table>

### Actions

The following actions can be perfomed on this resource.

<dl>
  <dt>Create</dt>
  <dd>POST /api/v1/terraform/configurations/:username/:name/versions</dd>
  <dt>Upload progress</dt>
  <dd>GET /api/v1/terraform/configurations/:username/:name/versions/progress/:token</dd>
</dl>

### Examples

#### Creating a configuration version

Creates a configuration with the provided attributes.

    $ cat version.json
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

    $ curl %{ATLAS_URL}/api/v1/terraform/configurations/%{DEFAULT_USERNAME}/test/versions \
        -X POST \
        -H "X-Atlas-Token: $ATLAS_TOKEN" \
        -H "Content-Type: application/json" \
        -d @version.json

#### Retrieving the progress of an upload for a configuration version

Returns upload progress for the version.

    $ curl %{ATLAS_URL}/api/v1/terraform/configurations/%{DEFAULT_USERNAME}/test/versions/progress/63fc7e18-3911-4853-8b17-7fdc28f158f2 \
        -H "X-Atlas-Token: $ATLAS_TOKEN"
