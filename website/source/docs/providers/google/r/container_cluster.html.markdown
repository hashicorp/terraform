---
layout: "google"
page_title: "Google: google_container_cluster"
sidebar_current: "docs-google-container-cluster"
description: |-
  Creates a GKE cluster.
---

# google\_container\_cluster

-> **Note:** Due to limitations of the API, all arguments except `node_version` are non-updateable (changing any will cause recreation of the whole cluster).

## Example usage

```
resource "google_container_cluster" "primary" {
    name = "marcellus-wallace"
    zone = "us-central1-a"
    initial_node_count = 3

    master_auth {
        username = "mr.yoda"
        password = "adoy.rm"
    }

    node_config {
        oauth_scopes = [
            "https://www.googleapis.com/auth/compute",
            "https://www.googleapis.com/auth/devstorage.read_only",
            "https://www.googleapis.com/auth/logging.write",
            "https://www.googleapis.com/auth/monitoring"
        ]
    }
}
```

## Argument Reference

* `name` - (Required) The name of the cluster, unique within the project and zone
* `zone` - (Required) The zone that all resources should be created in.
* `master_auth` - (Required) The authentication information for accessing the Kubernetes master
* `initial_node_count` - (Required) The number of nodes to create in this cluster (not including the Kubernetes master)
* `description` - (Optional) Description of the cluster
* `node_version` - (Optional) The Kubernetes version on the nodes. Only valid for upgrading of existing cluster.
  Defaults to latest version supported by the server.
* `cluster_ipv4_cidr` - (Optional) The IP address range of the container pods in this cluster.
  Default is an automatically assigned CIDR.
* `logging_service` - (Optional) The logging service that the cluster should write logs to.
  Available options include `logging.googleapis.com` and `none`. Defaults to `logging.googleapis.com`
* `monitoring_service` - (Optional) The monitoring service that the cluster should write metrics to.
  Available options include `monitoring.googleapis.com` and `none`. Defaults to `monitoring.googleapis.com`
* `network` - (Optional) The name of the Google Compute Engine network to which the cluster is connected
* `node_config` -  (Optional) The machine type and image to use for all nodes in this cluster

**Master Auth** supports the following arguments:

* `password` - The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint
* `username` - The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint

**Node Config** supports the following arguments:

* `machine_type` - (Optional) The name of a Google Compute Engine machine type.
  Defaults to `n1-standard-1`.
* `disk_size_gb` - (Optional) Size of the disk attached to each node, specified in GB.
  The smallest allowed disk size is 10GB. Defaults to 100GB.
* `oauth_scopes` - (Optional) The set of Google API scopes to be made available on all
  of the node VMs under the "default" service account. The following scopes are necessary
  to ensure the correct functioning of the cluster:

  * `https://www.googleapis.com/auth/compute`
  * `https://www.googleapis.com/auth/devstorage.read_only`
  * `https://www.googleapis.com/auth/logging.write` (if `logging_service` points to Google)
  * `https://www.googleapis.com/auth/monitoring` (if `monitoring_service` points to Google)

## Attributes Reference

* `master_auth.client_certificate` - Base64 encoded public certificate
  used by clients to authenticate to the cluster endpoint.
* `master_auth.client_key` - Base64 encoded private key used by clients
  to authenticate to the cluster endpoint
* `master_auth.cluster_ca_certificate` - Base64 encoded public certificate
  that is the root of trust for the cluster
* `endpoint` - The IP address of this cluster's Kubernetes master
* `instance_group_urls` - List of instance group URLs which have been assigned to the cluster
