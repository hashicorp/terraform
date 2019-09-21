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

Optionally stores the lock as another artifact in a repository in Artifactory.

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
    # optionally add the followings to enable locking
    lock_username   = "LockUser"
    lock_password   = "LockPass"
    unlock_username = "UnlockUser"
    unlock_password = "UnlockPass"
    lock_url        = "https://custom.artifactoryonline.com/artifactory"
    lock_repo       = "foo-lock"
    lock_subpath    = "terraform-bar-lock"
    lock_readback_wait = 200
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
    lock_subpath    = "terraform-bar-lock"
    lock_readback_wait = 200

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
 * `lock_username` / `LOCK_ARTIFACTORY_USERNAME` (Optional) - The username to create the lock artifact with
 * `lock_password` / `LOCK_ARTIFACTORY_PASSWORD` (Optional) - The password to create the lock artifact with
 * `unlock_username` / `UNLOCK_ARTIFACTORY_USERNAME` (Optional) - The username to remove the lock artifact with
 * `unlock_password` / `UNLOCK_ARTIFACTORY_PASSWORD` (Optional) - The password to remove the lock artifact with
 * `lock_url` / `LOCK_ARTIFACTORY_URL` (Optional) - The base url for lock artifact access. `lock_url` and `url` should be the same when the state and lock artifact are stored on the same artifactory server
 * `lock_repo` (Optional) - The repository name for lock artifact access
 * `lock_subpath` (Optional) - Path within `lock_repo` for lock artifact access
 * `lock_readback_wait` (Optional) - After a lock file is put into artifactory, it can be optionally readback after `lock_readback_wait` milliseconds from artifactory to compare with the orignal lock file. This is to make sure that the lock is properly created in artifactory. When value <=0 or not specified, no readback is performed. Empirical recommandation: >=200.

## Repo configuration

Repo configuration might be different, depending on the storage server type.

### Jfrog Artifactory
* `username` should have `Delete/Overwrite`, `Deploy/Cache` and `Read` permission on `repo`
* `lock_username` should have `Deploy/Cache` and `Read` permission, but not `Delete/Overwrite` permission on `lock_repo`
* `unlock_username` should have `Delete/Overwrite` permission on `lock_repo`
* [reference](https://www.jfrog.com/confluence/display/RTF/Managing+Permissions)

### Sonatype Nexus:
* `repo`'s "Deployment Policy" should "Allow Redeploy"
* `lock_repo`'s "Deployment Policy" should "Disable Redeploy" 
* `unlock_username` and `unlock_password` should be omitted
* [reference](https://help.sonatype.com/repomanager2/configuration/managing-repositories)
