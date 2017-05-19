---
layout: "github"
page_title: "GitHub: github_repository_deploy_key"
sidebar_current: "docs-github-resource-repository-deploy-key"
description: |-
  Provides a GitHub repository deploy key resource.
---

# github_repository_deploy_key

Provides a GitHub repository deploy key resource. A deploy key is an SSH key that is stored on your server and grants
access to a single GitHub repository. This key is attached directly to the repository instead of to a personal user
account.

This resource allows you to add/remove repository deploy keys.

Further documentation on GitHub repository deploy keys:

- [About deploy keys](https://developer.github.com/guides/managing-deploy-keys/#deploy-keys)

## Example Usage

```hcl
# Add a deploy key
resource "github_repository_deploy_key" "example_repository_deploy_key" {
	title = "Repository test key"
	repository = "test-repo"
	key = "ssh-rsa AAA..."
	read_only = "false"
}
```

## Argument Reference

The following arguments are supported:

* `key` - (Required) A ssh key.
* `read_only` - (Required) A boolean qualifying the key to be either read only or read/write.
* `repository` - (Required) Name of the Github repository.
* `title` - (Required) A title.

Changing any of the fields forces re-creating the resource.
