---
layout: "remotestate"
page_title: "Remote State Backend: etcd"
sidebar_current: "docs-state-remote-etcd"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# etcd

Stores the state in [etcd](https://coreos.com/etcd/) at a given path.

## Example Usage

```
terraform remote config \
	-backend=etcd \
	-backend-config="path=path/to/terraform.tfstate" \
	-backend-config="endpoints=http://one:4001 http://two:4001"
```

## Example Referencing

```
resource "terraform_remote_state" "foo" {
	backend = "etcd"
	config {
		path = "path/to/terraform.tfstate"
		endpoints = "http://one:4001 http://two:4001"
	}
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Required) The path where to store the state
 * `endpoints` - (Required) A space-separated list of the etcd endpoints
 * `username` - (Optional) The username
 * `password` - (Optional) The password
