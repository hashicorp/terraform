---
layout: "remotestate"
page_title: "Remote State Backend: github"
sidebar_current: "docs-state-remote-github"
description: |-
  Terraform can store the state in a GitHub repository, making it easier to
  version and work with in a team.
---

# GitHub

Stores the state in a file in a [GitHub](https://github.com) repository.

## Using GitHub for Remote State

To enable remote state in a GitHub repository we run the `terraform
remote config` command like so:

```
terraform remote config \
	-backend=github \
	-backend-config="owner=orgname_or_username" \
	-backend-config="repository=my_repo" \
	-backend-config="state_path=my_state_file" \
	-backend-config="lock_path=my_lock_file"
```

This assumes that the organization/user has a repository called
`my_repo`. The Terraform state is written to the file `my_state_file`
in that repository.

-> **Note:** Passing credentials directly via configuration options will
make them included in cleartext inside the persisted state. Use of
environment variables or a configuration file is recommended.

## Using the GitHub remote state

To make use of the GitHub remote state we can use the
[`terraform_remote_state` data
source](/docs/providers/terraform/d/remote_state.html).

```
data "terraform_remote_state" "foo" {
	backend = "github"
	config {
    owner = "username"
    token = "861d16f0809f2ed0ceb51f29e8391995cd5025de"
    repository = "my-repo"
    state_file = "my_state_file"
	}
}
```

## Configuration variables

The following configuration options or environment variables are supported:

 * `owner` - (Required) The organization name or user name that owns the
    repository.
 * `token` - (Required) A personal access token (that confers the right to
     read/write/delete files in the repository)
 * `repository` - (Required) The name of the repository.
 * `state_path` - (Required) The name of the file in which to save the state.
 * `lock_path` - (Required) The name of the file to use for locking.
 * `branch` - The name of the branch in which to work.
 * `base_url` - The GitHub API URL (supports GitHubEnterprise).
