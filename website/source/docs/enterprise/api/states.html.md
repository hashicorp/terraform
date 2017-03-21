---
layout: "api"
page_title: "State API"
sidebar_current: "docs-enterprise-api-states"
description: |-
  State represents the status of your infrastructure at the last time Terraform was run.
---

# State API

State represents the status of your infrastructure at the last time Terraform was run. States can be pushed to Terraform Enterprise from Terraform's CLI after an apply is done locally, or state is automatically stored if the apply is done in Terraform Enterprise.

### State Attributes

<table>
  <tr>
    <th>Attribute</th>
    <th>Description</th>
    <th>Required</th>
  </tr>
  <tr>
    <td><code>username</code></td>
    <td>If supplied, only return states belonging to the organization with this username.</td>
    <td>No</td>
  </tr>
</table>

### Actions

The following actions can be perfomed on this resource.

<dl>
  <dt>Get a list of states accessible to a user</dt>
  <dd>GET /api/v1/terraform/state</dd>
</dl>

### Examples

#### Getting a list of Terraform states

    $ curl %{ATLAS_URL}/api/v1/terraform/state \
        -H "X-Atlas-Token: $ATLAS_TOKEN"

#### Getting a list of Terraform states for an organization

    $ curl %{ATLAS_URL}/api/v1/terraform/state?username=acme_inc \
        -H "X-Atlas-Token: $ATLAS_TOKEN"

#### Getting second page of list of Terraform states

    $ curl %{ATLAS_URL}/api/v1/terraform/state?page=2 \
        -H "X-Atlas-Token: $ATLAS_TOKEN"
