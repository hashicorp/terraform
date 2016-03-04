---
layout: "remotestate"
page_title: "Remote State Backend: artifactory"
sidebar_current: "docs-state-remote-artifactory"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# artifactory

Stores the state as an artifact in a given repository in [Artifactory](https://www.jfrog.com/artifactory/).

Generic HTTP repositories are supported, and state from different
configurations may be kept at different subpaths within the repository.

-> **Note:** The URL must include the path to the Artifactory installation.
It will likely end in `/artifactory`.

## Example Usage

```
terraform remote config \
	-backend=artifactory \
	-backend-config="username=SheldonCooper" \
	-backend-config="password=AmyFarrahFowler" \
	-backend-config="url=https://custom.artifactoryonline.com/artifactory" \
	-backend-config="repo=foo" \
	-backend-config="subpath=terraform-bar"
```

## Example Referencing

```
resource "terraform_remote_state" "foo" {
	backend = "artifactory"
	config {
		username = "SheldonCooper"
		password = "AmyFarrahFowler"
		url = "https://custom.artifactoryonline.com/artifactory"
		repo = "foo"
		subpath = "terraform-bar"
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
