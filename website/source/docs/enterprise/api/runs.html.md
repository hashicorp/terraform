---
title: "Runs API"
---

# Runs API

Runs in Atlas represents a two step Terraform plan and a subsequent apply.

Runs are queued under [environments](/help/api/terraform/environments)
and require a two-step confirmation workflow. However, environments
can be configured to auto-apply to avoid this.

### Run Attributes

<table>
  <tr>
    <th>Attribute</th>
    <th>Description</th>
    <th>Required</th>
  </tr>
  <tr>
    <td><code>destroy</code></td>
    <td>If set to <code>true</code>, this run will be a destroy plan.</td>
    <td>No</td>
  </tr>
</table>

### Actions

The following actions can be perfomed on this resource.

<dl>
  <dt>Queue a run</dt>
  <dd>POST /api/v1/enviromments/:username/:name/plan</dd>
</dl>

### Examples

#### Queueing a new run

Starts a new run (plan) in the environment. Requires a configuration
version to be present on the environment to succeed, but will otherwise 404.

    $ curl %{ATLAS_URL}/api/v1/environments/%{DEFAULT_USERNAME}/test/plan \
        -X POST \
        -d "" \
        -H "X-Atlas-Token: $ATLAS_TOKEN"
