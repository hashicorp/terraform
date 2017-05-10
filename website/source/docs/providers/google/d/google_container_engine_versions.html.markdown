---
layout: "google"
page_title: "Google: google_container_engine_versions"
sidebar_current: "docs-google-datasource-container-versions"
description: |-
  Provides lists of available Google Container Engine versions for masters and nodes.
---

# google\_container\_engine\_versions

Provides access to available Google Container Engine versions in a zone for a given project.

```hcl
data "google_container_engine_versions" "central1b" {
  zone = "us-central1-b"
}

resource "google_container_cluster" "foo" {
  name               = "terraform-test-cluster"
  zone               = "us-central1-b"
  node_version       = "${data.google_container_engine_versions.central1b.latest_node_version}"
  initial_node_count = 1

  master_auth {
    username = "mr.yoda"
    password = "adoy.rm"
  }
}
```

## Argument Reference

The following arguments are supported:

* `zone` (required) - Zone to list available cluster versions for. Should match the zone the cluster will be deployed in.
* `project` (optional) - ID of the project to list available cluster versions for. Should match the project the cluster will be deployed to.
  Defaults to the project that the provider is authenticated with.

## Attributes Reference

The following attributes are exported:

* `valid_master_versions` - A list of versions available in the given zone for use with master instances.
* `valid_node_versions` - A list of versions available in the given zone for use with node instances.
* `latest_master_version` - The latest version available in the given zone for use with master instances.
* `latest_node_version` - The latest version available in the given zone for use with node instances.
