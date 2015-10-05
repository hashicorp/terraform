---
layout: "Template"
page_title: "Template: cloudinit_multipart"
sidebar_current: "docs-template-resource-cloudinit-config"
description: |-
  Renders a cloud-init config.
---

# template\_cloudinit\_config

Renders a template from a file.

## Example Usage

```
resource "template_file" "script" {
    template = "${file("${path.module}/init.tpl")}"

    vars {
        consul_address = "${aws_instance.consul.private_ip}"
    }
}

resource "template_cloudinit_config" "config" {
    # Setup hello world script to be called by the cloud-config
    part {
        filename = "init.cfg"
        content_type = "text/part-handler"
        content = "${template_file.script.rendered}"
    }

    # Setup cloud-config yaml
    part {
        content_type = "text/cloud-config"
        content = "${file(\"config.yaml\")"
    }
}



```

## Argument Reference

The following arguments are supported:

* `gzip` - (Optional) Specify whether or not to gzip the rendered output.

* `base64_encode` - (Optional) Base64 encoding of the rendered output.

* `part` - (Required) One may specify this many times, this creates a fragment of the rendered cloud-init config.

The `part` block supports:

* `filename` - (Optional) Filename to save part as.

* `content_type` - (Optional) Content type to send file as.

* `content` - (Required) Body for the part.

* `merge_type` - (Optional) Gives the ability to merge multiple blocks of cloud-config together.


## Attributes Reference

The following attributes are exported:

* `rendered` - The final rendered template.
