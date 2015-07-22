---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_replication_controller"
sidebar_current: "docs-kubernetes-resource-replication-controller"
description: |-
  Creates a Kubernetes Replication Controller.
---

# kubernetes\_replication\_controller

A replication controller ensures that a specified number of pod "replicas"
are running at any one time. If there are too many, it will kill some.
If there are too few, it will start more.

See [documentation](http://kubernetes.io/v1.0/docs/user-guide/replication-controller.html#what-is-a-replication-controller) for details.


## Example Usage

```
resource "kubernetes_replication_controller" "wp" {
    name = "wordpress"
    labels {
        Name = "WordPress"
    }
    spec = <<SPEC
replicas: 2
selector:
  app: nginx
template:
  metadata:
    labels:
      app: nginx
  spec:
    containers:
    - name: nginx
      image: nginx
      ports:
      - containerPort: 80
SPEC
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the replication controller.
* `spec` - (Required) The specification of the replication controller. Only `spec` section of
    the YAML or JSON structure is required (i.e. `replicas`, `selector` & `template` will be on the root level).
    See [documentation](http://kubernetes.io/v1.0/docs/user-guide/walkthrough/k8s201.html#replication-controllers) for details.
* `namespace` - (Optional) Namespace defines the space within which `name` must be unique.
    See [documentation](https://github.com/GoogleCloudPlatform/kubernetes/blob/v1/docs/design/namespaces.md)
    for details about other effects & features of namespaces.
* `labels` - (Optional) A list of labels attached to the replication controller.

## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the replication controller

