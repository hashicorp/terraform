---
layout: "backend-types"
page_title: "Backend Type: artifactory"
sidebar_current: "docs-backends-types-standard-artifactory"
description: |-
  Terraform can store state in artifactory.
---

# artifactory

**Kind: Standard (with no locking)**

Stores the state as an artifact in a given repository in
[Artifactory](https://www.jfrog.com/artifactory/).

Generic HTTP repositories are supported, and state from different
configurations may be kept at different subpaths within the repository.

-> **Note:** The URL must include the path to the Artifactory installation.
It will likely end in `/artifactory`.

## Example Configuration

```hcl
terraform {
  backend "artifactory" {
    username = "SheldonCooper"
    password = "AmyFarrahFowler"
    url      = "https://custom.artifactoryonline.com/artifactory"
    repo     = "foo"
    subpath  = "teraraform-bar"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "artifactory"
  config = {
    username = "SheldonCooper"
    password = "AmyFarrahFowler"
    url      = "https://custom.artifactoryonline.com/artifactory"
    repo     = "foo"
    subpath  = "terraform-bar"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `username` / `ARTIFACTORY_USERNAME` (Required) - The username
 * `password` / `ARTIFACTORY_PASSWORD` (Required) - The password
 * `url` / `ARTIFACTORY_URL` (Required) - The URL. Note that this is the base url to artifactory not the full repo and subpath.
 * `repo` (Required) - The repository name
 * `subpath` (Required) - Path within the repository
