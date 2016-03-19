---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_persistent_volume"
sidebar_current: "docs-kubernetes-resource-persistent-volume"
description: |-
  Creates a Kubernetes Persistent Volume.
---

# kubernetes\_volume

Persistent volumes VS volume claims

http://kubernetes.io/v1.0/docs/user-guide/persistent-volumes/README.html
https://github.com/kubernetes/kubernetes/blob/master/docs/user-guide/persistent-volumes/claims/claim-01.yaml
https://github.com/kubernetes/kubernetes/blob/v1.0.3/docs/user-guide/volumes.md

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
* `labels` - (Optional) A list of labels attached to the volume.
* `spec` - (Required) The specification of the volume. Only `spec` section of
    the YAML or JSON structure is required (i.e. `capacity`, `accessModes` & `hostPath` will be on the root level).
    See [documentation](http://kubernetes.io/v1.0/docs/user-guide/persistent-volumes/README.html) for details.
