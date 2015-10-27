---
layout: "docs"
page_title: "Provisioners: null_resource"
sidebar_current: "docs-provisioners-null-resource"
description: |-
  The `null_resource` is a resource allows you to configure provisioners that
  are not directly associated with a single exiting resource.
---

# null\_resource

The `null_resource` is a resource that allows you to configure provisioners
that are not directly associated with a single existing resource.

A `null_resource` behaves exactly like any other resource, so you configure
[provisioners](/docs/provisioners/index.html), [connection
details](/docs/provisioners/connection.html), and other meta-parameters in the
same way you would on any other resource.

This allows fine-grained control over when provisioners run in the dependency
graph.

## Example usage

```
# Bootstrap a cluster after all its instances are up
resource "aws_instance" "cluster" {
  count = 3
  // ...
}

resource "null_resource" "cluster" {
  # Changes to any instance of the cluster requires re-provisioning
  triggers {
    cluster_instance_ids = "${join(",", aws_instance.cluster.*.id)}"
  }

  # Bootstrap script can run on any instance of the cluster
  # So we just choose the first in this case
  connection {
    host = "${element(aws_instance.cluster.*.public_ip, 0)}"
  }

  provisioner "remote-exec" {
    # Bootstrap script called with private_ip of each node in the clutser
    inline = [
      "bootstrap-cluster.sh ${join(" ", aws_instance.cluster.*.private_ip}"
    ]
  }
}
```

## Argument Reference

In addition to all the resource configuration available, `null_resource` supports the following specific configuration options:

 * `triggers` - A mapping of values which should trigger a rerun of this set of
   provisioners. Values are meant to be interpolated references to variables or
   attributes of other resources.

