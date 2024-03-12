---
layout: "language"
page_title: "Backend Type: nxrm"
sidebar_current: "docs-backends-types-standard-nxrm"
description: |-
  Terraform can store state remotely in an NXRM instance. Provides state locking to prevent clashes amongst team members.
---

# NXRM

**Kind: Standard (with locking)**

Stores the state as an artifact in the specified repository in
[NXRM](https://help.sonatype.com/repomanager3).

Generic HTTP repositories are supported, and state from different
configurations may be kept at different subpaths within the repository.

-> **Note:** The URL must include the path to the NXRM installation as well as the name of your custom repo.
The NXRM installation path will likely be `/repository`.

## Example Configuration

```hcl
terraform {
  backend "nxrm" {
    username   = "Morty"
    password   = "NotRick"
    url        = "https://nxrm.example.com/repository/tf-repo"
    subpath    = "my-tf-state"
    state_name = "terraform.tfstate"
    timeout    = 30
  }
}
```

## Data Source Configuration

```hcl
data "terraform_remote_state" "foo" {
  backend = "nxrm"
  config = {
    username   = "Morty"
    password   = "NotRick"
    url        = "https://nxrm.example.com/repository/tf-repo"
    subpath    = "my-tf-state"
    state_name = "terraform.tfstate"
    timeout    = 30
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `username` / `NXRM_USERNAME` (Required) - The username
 * `password` / `NXRM_PASSWORD` (Required) - The password
 * `url` / `NXRM_URL` (Required) - The URL. Note that this is the base url to NXRM plus the custom repository.
 * `subpath` / `NXRM_SUBPATH` (Required) - Path within the repository
 * `timeout` / `NXRM_CLIENT_TIMEOUT` - The timeout to set on the connection (defaults to 30 seconds if not specified).
