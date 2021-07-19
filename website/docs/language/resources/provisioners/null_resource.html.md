---
layout: "language"
page_title: "Provisioners Without a Resource"
sidebar_current: "docs-provisioners-null-resource"
description: "'null_resource' allows you to configure provisioners that are not directly associated with a single existing resource."
---

# Provisioners Without a Resource

[null]: https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource

If you need to run provisioners that aren't directly associated with a specific
resource, you can associate them with a `null_resource`.

Instances of [`null_resource`][null] are treated like normal resources, but they
don't do anything. Like with any other resource, you can configure
[provisioners](/docs/language/resources/provisioners/syntax.html) and [connection
details](/docs/language/resources/provisioners/connection.html) on a `null_resource`. You can also
use its `triggers` argument and any meta-arguments to control exactly where in
the dependency graph its provisioners will run.

## Example usage

```hcl
resource "aws_instance" "cluster" {
  count = 3

  # ...
}

resource "null_resource" "cluster" {
  # Changes to any instance of the cluster requires re-provisioning
  triggers = {
    cluster_instance_ids = "${join(",", aws_instance.cluster.*.id)}"
  }

  # Bootstrap script can run on any instance of the cluster
  # So we just choose the first in this case
  connection {
    host = "${element(aws_instance.cluster.*.public_ip, 0)}"
  }

  provisioner "remote-exec" {
    # Bootstrap script called with private_ip of each node in the cluster
    inline = [
      "bootstrap-cluster.sh ${join(" ", aws_instance.cluster.*.private_ip)}",
    ]
  }
}
```

## Argument Reference

In addition to meta-arguments supported by all resources, `null_resource`
supports the following specific arguments:

 * `triggers` - A map of values which should cause this set of provisioners to
   re-run. Values are meant to be interpolated references to variables or
   attributes of other resources.
