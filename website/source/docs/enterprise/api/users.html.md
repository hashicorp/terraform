---
layout: "enterprise"
page_title: "Users - API - Terraform Enterprise"
sidebar_current: "docs-enterprise-api-users"
description: |-
  Users are both users and organizations in Terraform Enterprise. They are the parent resource of all resources.
---

# Users API

Users are both users and organizations in Terraform Enterprise. They are the
parent resource of all resources.

Currently, only the retrieval of users is available on the API. Additionally,
only Vagrant box resources will be listed. Boxes will be returned based on
permissions over the organization, or user.

## Read User

This endpoint retrieves information about a single user.

| Method | Path           |
| :----- | :------------- |
| `GET`  | `/user/:username` |

### Parameters

- `:username` `(string: <required>)` - Specifies the username to search. This is
  specified as part of the URL.

### Sample Request

```text
$ curl \
    --header "X-Atlas-Token: ..." \
    https://atlas.hashicorp.com/api/v1/user/my-user
```

### Sample Response

```json
{
  "username": "sally-seashell",
  "avatar_url": "https://www.gravatar.com/avatar/...",
  "profile_html": "Sally is...",
  "profile_markdown": "Sally is...",
  "boxes": []
}
```
