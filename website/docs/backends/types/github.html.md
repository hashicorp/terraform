---
layout: "backend-types"
page_title: "Backend Type: github"
sidebar_current: "docs-backends-types-standard-github"
description: |-
  Terraform can store the state in a GitHub repository.
---

# GitHub

**Kind: Standard (with locking)**

Stores the state in a file in a [GitHub](https://github.com) repository.

This backend supports [state locking](/docs/state/locking.html).

## Example Configuration

Here's a sample configuration that stores state in the
`terraform-state` branch of the repository at
`https://github.mycompany.com/snoopy/terraform-configs`.

```hcl
terraform {
  backend "github" {
    base_url = "https://github.mycompany.com/api/v3/"
    owner = "snoopy"
    repository = "terraform-configs"
    branch = "terraform-state"
  }
}
```

It is a [partial
configuration](/docs/backends/config.html#partial-configuration); in
particular the GitHub API token must be passed in separately (perhaps
via the `GITHUB_TOKEN` environment variable).

It requires that that the repository exists and has a
`terraform-state` branch; this is a good use for an orphan branch set
up something like so:

```sh
git checkout --orphan terraform-state
git rm -rf .
echo "# Terraform remote state" > README.md
echo "**HANDS OFF**" >> README.md
git add README.md
git commit -a -m "Initial commit"
git push origin terraform-state
```

## Example Referencing

```
data "terraform_remote_state" "foo" {
    backend = "github"
    config {
        base_url = "https://github.mycompany.com/api/v3/"
        owner = "snoopy"
        repository = "terraform-configs"
        branch = "terraform-state"
    }
}
```

## Configuration variables

The following configuration options or environment variables are supported:

 * `owner` - (Required) The organization name or user name that owns the
    repository.
 * `repository` - (Required) The name of the repository (must already exist).
 * `token` / `GITHUB_TOKEN` - (Required) A personal access token (that
    confers the right to read/write/delete files in the repository)
 * `state_path` - The name of the file in which to save the state
    (default = terraform.tfstate).
 * `lock_path` - The name of the file to use for locking (default =
    terraform.tfstate.lock).
 * `branch` - The name of the branch in which to work (must already exist).
 * `base_url` - The GitHub API URL (supports GitHub Enterprise).
