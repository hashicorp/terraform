---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_persistent_volume_claim"
sidebar_current: "docs-kubernetes-resource-persistent-volume-claim"
description: |-
  Creates a Kubernetes Persistent Volume Claim.
---

# kubernetes\_persistent\_volume\_claim

A Persistent Volume Claim is a request for storage. It is similar to a [pod](/docs/providers/kubernetes/r/pod.html). Pods consume node resources and Volume Claims consume [Persistent Volume](/docs/providers/kubernetes/r/persistent_volume.html) resources. Pods can request specific levels of resources (CPU and Memory). Claims can request specific size and access modes (e.g, can be mounted once read/write or many times read-only).

Read more about Persistent Volume Claims [in docs](http://kubernetes.io/v1.0/docs/user-guide/persistent-volumes.html#persistentvolumeclaims).

## Example Usage

```
resource "kubernetes_persistent_volume_claim" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
accessModes:
  - ReadWriteOnce
resources:
  requests:
    storage: 8Gi
SPEC
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the volume claim.
* `spec` - (Required) The specification of the volume claim. Only `spec` section of
    the YAML or JSON structure is required (i.e. `accessModes` & `resources` will be on the root level).
    See [documentation](http://kubernetes.io/v1.0/docs/user-guide/persistent-volumes/README.html) for details.
* `namespace` - (Optional) Namespace defines the space within which `name` must be unique.
    See [documentation](https://github.com/GoogleCloudPlatform/kubernetes/blob/v1/docs/design/namespaces.md)
    for details about other effects & features of namespaces.
* `labels` - (Optional) A list of labels attached to the volume claim.

## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the volume claim.

