---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_persistent_volume"
sidebar_current: "docs-kubernetes-resource-persistent-volume"
description: |-
  Creates a Kubernetes Persistent Volume.
---

# kubernetes\_persistent\_volume

A Persistent Volume is a piece of networked storage in the cluster. It is a resource in the cluster just like a node is a cluster resource. This object captures the details of the implementation of the storage, be that NFS, iSCSI, or a cloud-provider-specific storage system.

Read more about Persistent Volumes [in docs](http://kubernetes.io/v1.0/docs/user-guide/persistent-volumes.html).

## Example Usage

```
resource "kubernetes_persistent_volume" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
capacity:
  storage: 10Gi
accessModes:
  - ReadWriteOnce
hostPath:
  path: "/tmp/data01"
SPEC
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the volume.
* `spec` - (Required) The specification of the volume. Only `spec` section of
    the YAML or JSON structure is required (i.e. `capacity`, `accessModes` & `hostPath` will be on the root level).
    See [documentation](http://kubernetes.io/v1.0/docs/user-guide/persistent-volumes/README.html) for details.
* `namespace` - (Optional) Namespace defines the space within which `name` must be unique.
    See [documentation](https://github.com/GoogleCloudPlatform/kubernetes/blob/v1/docs/design/namespaces.md)
    for details about other effects & features of namespaces.
* `labels` - (Optional) A list of labels attached to the volume.

## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the volume.

