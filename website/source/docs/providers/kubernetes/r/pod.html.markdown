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
    spec = <<SPEC
containers:
  - image: wordpress
    name: wordpress
    env:
      - name: WORDPRESS_DB_PASSWORD
        # change this - must match mysql.yaml password
        value: yourpassword
    ports:
      - containerPort: 80
        name: wordpress
    volumeMounts:
        # name must match the volume name below
      - name: wordpress-persistent-storage
        # mount path within the container
        mountPath: /var/www/html
volumes:
  - name: wordpress-persistent-storage
    gcePersistentDisk:
      # This GCE PD must already exist.
      pdName: wordpress-disk
      fsType: ext4
SPEC
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
