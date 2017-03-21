---
layout: "api"
page_title: "Users API"
sidebar_current: "docs-enterprise-api-users"
description: |-
  Users are both users and organizations in Terraform Enterprise. They are the parent resource of all resources.
---

# Users API

Users are both users and organizations in Terraform Enterprise. They are the
parent resource of all resources.

Currently, only the retrieval of users is avaiable on the API. Additionally,
only [box](/help/api/vagrant/boxes) resources will be listed. Boxes will
be returned based on permissions over the organization, or user.

### Actions

The following actions can be perfomed on this resource.

<dl>
  <dt>Show</dt>
  <dd>GET /api/v1/user/:username</dd>
</dl>

### Examples

#### Retrieve a user

    $ curl %{ATLAS_URL}/api/v1/user/%{DEFAULT_USERNAME} \
        -H "X-Atlas-Token: $ATLAS_TOKEN"
