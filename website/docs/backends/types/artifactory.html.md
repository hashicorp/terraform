---
layout: "backend-types"
page_title: "Backend Type: artifactory"
sidebar_current: "docs-backends-types-standard-artifactory"
description: |-
  Terraform can store state in artifactory.
---

# artifactory

**Kind: Standard (with optional locking)**

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
    username        = "SheldonCooper"
    password        = "AmyFarrahFowler"
    url             = "https://custom.artifactoryonline.com/artifactory"
    repo            = "foo"
    subpath         = "terraform-bar"
    # add the following to enable locking
    lock_username   = "LockUser"
    lock_password   = "LockPass"
    unlock_username = "UnlockUser"
    unlock_password = "UnlockPass"
    lock_url        = "https://custom.artifactoryonline.com/artifactory"
    lock_repo       = "foo-lock"
    lock_subpath    = "erraform-bar-lock"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "artifactory"
  config = {
    username        = "SheldonCooper"
    password        = "AmyFarrahFowler"
    url             = "https://custom.artifactoryonline.com/artifactory"
    repo            = "foo"
    subpath         = "terraform-bar"
    lock_username   = "LockUser"
    lock_password   = "LockPass"
    unlock_username = "UnlockUser"
    unlock_password = "UnlockPass"
    lock_url        = "https://custom.artifactoryonline.com/artifactory"
    lock_repo       = "foo-lock"
    lock_subpath    = "erraform-bar-lock"
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
 * `lock_username` / `LOCK_ARTIFACTORY_USERNAME` (Optional) - The username to create the lock file with
 * `lock_password` / `LOCK_ARTIFACTORY_PASSWORD` (Optional) - The password to create the lock file with
 * `unlock_username` / `UNLOCK_ARTIFACTORY_USERNAME` (Optional) - The username to remove the lock file with
 * `unlock_password` / `UNLOCK_ARTIFACTORY_PASSWORD` (Optional) - The password to remove the lock file with
 * `lock_url` / `LOCK_ARTIFACTORY_URL` (Optional) - The base url for lock file access. `lock_url` and `url` should be the same when the state and lock files are stored on the same artifactory server
 * `lock_repo` (Optional) - The repository name for lock file access
 * `lock_subpath` (Optional) - Path within `lock_repo` for lock file access

## Repo configuration

Repo configuration might be different, depending on the storage server type.

### Jfrog Artifactory
* `username` should have `Delete/Overwrite` and `Deploy/Cache` permission on `repo`
* `lock_username` should have `Deploy/Cache` permission, but not `Delete/Overwrite` permission on `lock_repo`
* `unlock_username` should have `Delete/Overwrite` permission on `lock_repo`
* [reference ] (https://www.jfrog.com/confluence/display/RTF/Managing+Permissions)

### Sonatype Nexus:
* the "Deployment Policy" of `repo` should "Allow Redeploy"
* the "Deployment Policy" of `lock_repo` should "Disable Redeploy" 
* `unlock_username` and `unlock_password` should be omitted
* [reference] (https://help.sonatype.com/repomanager2/configuration/managing-repositories)
