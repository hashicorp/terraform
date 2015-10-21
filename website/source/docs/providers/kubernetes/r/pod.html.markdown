---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_pod"
sidebar_current: "docs-kubernetes-resource-pod"
description: |-
  Creates a Kubernetes Pod.
---

# kubernetes\_pod

In Kubernetes, rather than individual application containers,
pods are the smallest deployable units that can be created, scheduled, and managed.
See [documentation](http://kubernetes.io/v1.0/docs/user-guide/pods.html) for details.


## Example Usage

```
resource "kubernetes_pod" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }

    container {
        image = "redis"
        name = "redis-master"

        port {
            container_port = 6379
            protocol = "TCP"
        }

        volume_mount = {
            name = "empty"
            mount_path = "/"
        }

        image_pull_policy = "Always"
    }

    volume {
        name = "empty"
        volume_source {
            empty_dir {
                medium = "Memory"
            }
        }
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the pod.
* `spec` - (Required) The specification of the pod. Only `spec` section of
    the YAML or JSON structure is required (i.e. `containers` & `volumes` will be on the root level).
    See [documentation](http://kubernetes.io/v1.0/docs/user-guide/walkthrough/README.html#pods) for details.
* `namespace` - (Optional) Namespace defines the space within which `name` must be unique.
    See [documentation](https://github.com/GoogleCloudPlatform/kubernetes/blob/v1/docs/design/namespaces.md)
    for details about other effects & features of namespaces.
* `labels` - (Optional) A list of labels attached to the pod.

## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the pod
