---
layout: "bitbucket"
page_title: "Provider: Bitbucket"
sidebar_current: "docs-bitbucket-index"
description: |-
  The Bitbucket proivder to interact with repositories, projects, etc..
---

# Bitbucket Provider

The Bitbucket provider allows you to manage resources including repositories,
webhooks, and default reviewers.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Bitbucket Provider
provider "bitbucket" {
    username = "GobBluthe"
    password = "idoillusions" # you can also use app passwords
}

resource "bitbucket_repsitory" "illusions" {
    owner = "theleagueofmagicians"
    name = "illussions"
    scm = "hg"
    is_private = true
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `username` - (Required) Your username used to connect to bitbucket. You can
  also set this via the environment variable. `BITBUCKET_USERNAME`

* `username` - (Required) Your passowrd used to connect to bitbucket. You can
  also set this via the environment variable. `BITBUCKET_PASSWORD`
