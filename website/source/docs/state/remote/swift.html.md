---
layout: "remotestate"
page_title: "Remote State Backend: swift"
sidebar_current: "docs-state-remote-swift"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# swift

Stores the state as an artifact in [Swift](http://docs.openstack.org/developer/swift/).

## Example Usage

```
terraform remote config \
	-backend=swift \
	-backend-config="path=random/path"
```

## Example Referencing

```
data "terraform_remote_state" "foo" {
	backend = "swift"
	config {
		path = "random/path"
	}
}
```

## Configuration variables

The following configuration option is supported:

 * `path` - (Required) The path where to store `terraform.tfstate`
 * `insecure` - (Optional) Allow "insecure" SSL requests. Defaults to `false`.

The following environment variables are supported:

 * `OS_AUTH_URL` - (Required) The identity endpoint
 * `OS_USERNAME` - (Required) The username
 * `OS_PASSWORD` - (Required) The password
 * `OS_REGION_NAME` - (Required) The region
 * `OS_TENANT_NAME` - (Required) The name of the tenant
 * `OS_DOMAIN_ID` - (Optional) The ID of the domain
 * `OS_DOMAIN_NAME` - (Optional) The name of the domain
