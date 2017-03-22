---
layout: "api"
page_title: "Environments API"
sidebar_current: "docs-enterprise-api-environments"
description: |-
  Environments represent running infrastructure managed by Terraform.
---

# Environments API

Environments represent running infrastructure managed by Terraform.

Environments can also be connected to Consul clusters.
This documentation covers the environment interactions with Terraform.

### Environment Attributes

<table class="apidocs">
  <tr>
    <th>Attribute</th>
    <th>Description</th>
    <th>Required</th>
  </tr>
  <tr>
    <td><code>variables</code></td>
    <td>A key/value map of Terraform variables to be updated. Existing
      variables will only be removed when their value is empty. Varaibles
      of the same key will be overwritten.</td>
    <td>Yes</td>
  </tr>
</table>
<br>
<div class="alert-infos">
  <div class="row alert-info">
    Note: Only string variables can be updated via the API currently.
    Creating or updating HCL variables is not yet supported.
  </div>
</div>

### Actions

The following actions can be perfomed on this resource.

<dl>
  <dt>Update variables</dt>
  <dd>PUT /api/v1/enviromments/:username/:name/variables</dd>
</dl>

### Examples

#### Updating Terraform variables

Updates the Terraform variables for an environment. Due to the sensitive nature
of variables, they will not returned on success.

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
