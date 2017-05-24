---
layout: "github"
page_title: "Github: github_user"
sidebar_current: "docs-github-datasource-user"
description: |-
  Get information on a Github user.
---

# github\_user

Use this data source to retrieve information about a Github user.

## Example Usage

```
data "github_user" "example" {
  username = "example"
}
```

## Argument Reference

 * `username` - (Required) The username.

## Attributes Reference

 * `login` - the user's login.
 * `avatar_url` - the user's avatar URL.
 * `gravatar_id` - the user's gravatar ID.
 * `site_admin` - whether the user is a Github admin.
 * `name` - the user's full name.
 * `company` - the user's company name.
 * `blog` - the user's blog location.
 * `location` - the user's location.
 * `email` - the user's email.
 * `bio` - the user's bio.
 * `public_repos` - the number of public repositories.
 * `public_gists` - the number of public gists.
 * `followers` - the number of followers.
 * `following` - the number of following users.
 * `created_at` - the creation date.
 * `updated_at` - the update date.

