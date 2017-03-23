---
layout: "google"
page_title: "Google: google_container_node_pool"
sidebar_current: "docs-google-container-node-pool"
description: |-
  Manages a GKE NodePool resource.
---

# google\_container\_node\_pool

Manages a Node Pool resource within GKE. For more information see
[the official documentation](https://cloud.google.com/container-engine/docs/node-pools)
and
[API](https://cloud.google.com/container-engine/reference/rest/v1/projects.zones.clusters.nodePools).

## Example usage

```tf
resource "google_container_node_pool" "np" {
  name               = "my-node-pool"
  zone               = "us-central1-a"
  cluster            = "${google_container_cluster.primary.name}"
  initial_node_count = 3
}

resource "google_container_cluster" "primary" {
  name               = "marcellus-wallace"
  zone               = "us-central1-a"
  initial_node_count = 3

  additional_zones = [
    "us-central1-b",
    "us-central1-c",
  ]

  master_auth {
    username = "mr.yoda"
    password = "adoy.rm"
  }

  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
  }
}
```

## Argument Reference

* `zone` - (Required) The zone in which the cluster resides.

* `cluster` - (Required) The cluster to create the node pool for.

* `initial_node_count` - (Required) The initial node count for the pool.

- - -

* `project` - (Optional) The project in which to create the node pool. If blank,
    the provider-configured project will be used.

* `name` - (Optional) The name of the node pool. If left blank, Terraform will
    auto-generate a unique name.

* `name_prefix` - (Optional) Creates a unique name for the node pool beginning
    with the specified prefix. Conflicts with `name`.

* `node_config` - (Optional) The machine type and image to use for all nodes in
    this pool

**Node Config** supports the following arguments:

* `machine_type` - (Optional) The name of a Google Compute Engine machine type.
    Defaults to `n1-standard-1`.

* `disk_size_gb` - (Optional) Size of the disk attached to each node, specified
    in GB. The smallest allowed disk size is 10GB. Defaults to 100GB.

* `local_ssd_count` - (Optional) The amount of local SSD disks that will be
    attached to each node pool. Defaults to 0.

* `oauth_scopes` - (Optional) The set of Google API scopes to be made available
    on all of the node VMs under the "default" service account. These can be
    either FQDNs, or scope aliases. The following scopes are necessary to ensure
    the correct functioning of the node pool:

  * `compute-rw` (`https://www.googleapis.com/auth/compute`)
  * `storage-ro` (`https://www.googleapis.com/auth/devstorage.read_only`)
  * `logging-write` (`https://www.googleapis.com/auth/logging.write`),
    if `logging_service` points to Google
  * `monitoring` (`https://www.googleapis.com/auth/monitoring`),
    if `monitoring_service` points to Google

* `service_account` - (Optional) The service account to be used by the Node VMs.
    If not specified, the "default" service account is used.

* `metadata` - (Optional) The metadata key/value pairs assigned to instances in
    the node pool.

* `image_type` - (Optional) The image type to use for this node.
