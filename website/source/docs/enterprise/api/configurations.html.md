---
title: "Terraform Configuration API"
---

# Terraform Configuration API

A configuration respresents settings associated with a resource that
runs Terraform with versions of Terraform configuration.

Configurations have many [configuration versions](/help/api/terraform/configuration-versions)
which represent versions of Terraform configuration templates and other associated
configuration.

### Configuration Attributes

<table>
  <tr>
    <th>Attribute</th>
    <th>Description</th>
    <th>Required</th>
  </tr>
  <tr>
    <td><code>name</code></td>
    <td>The name of the configuration, used to identify it. It
      has a maximum length of 50 characters and must contain only
      letters, numbers, dashes, underscores or periods.</td>
    <td>Yes</td>
  </tr>
  <tr>
    <td><code>username</code></td>
    <td>The username to assign the configuration to. You must be a member of the
      organization and have the ability to create the resource.</td>
    <td>Yes</td>
  </tr>
</table>

### Actions

The following actions can be perfomed on this resource.

<dl>
  <dt>Show</dt>
  <dd>GET /api/v1/terraform/configurations/:username/:name/versions/latest</dd>
  <dt>Create</dt>
  <dd>POST /api/v1/terraform/configurations</dd>
</dl>

### Examples

#### Creating a configuration

Creates a configuration with the provided attributes.

    $ curl %{ATLAS_URL}/api/v1/terraform/configurations \
        -X POST \
        -H "X-Atlas-Token: $ATLAS_TOKEN" \
        -d configuration[name]='test' \
        -d configuration[username]='%{DEFAULT_USERNAME}'

#### Retrieving a configuration

Returns the JSON respresentation of the latest configuration.

    $ curl %{ATLAS_URL}/api/v1/terraform/configurations/%{DEFAULT_USERNAME}/test/versions/latest \
        -H "X-Atlas-Token: $ATLAS_TOKEN"
