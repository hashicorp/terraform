---
layout: "Template"
page_title: "Template: cloudinit_multipart"
sidebar_current: "docs-template-resource-cloudinit-config"
description: |-
  Renders a multi-part cloud-init config from source files.
---

# template\_cloudinit\_config

Renders a multi-part cloud-init config from source files.

## Example Usage

```
# Render a part using a `template_file`
resource "template_file" "script" {
  template = "${file("${path.module}/init.tpl")}"

  vars {
    consul_address = "${aws_instance.consul.private_ip}"
  }
}

# Render a multi-part cloudinit config making use of the part
# above, and other source files
resource "template_cloudinit_config" "config" {
  gzip          = true
  base64_encode = true

  # Setup hello world script to be called by the cloud-config
  part {
    filename     = "init.cfg"
    content_type = "text/part-handler"
    content      = "${template_file.script.rendered}"
  }

  part {
    content_type = "text/x-shellscript"
    content      = "baz"
  }

  part {
    content_type = "text/x-shellscript"
    content      = "ffbaz"
  }
}

# Start an AWS instance with the cloudinit config as user data
resource "aws_instance" "web" {
  ami           = "ami-d05e75b8"
  instance_type = "t2.micro"
  user_data     = "${template_cloudinit_config.config.rendered}"
}
```

## Argument Reference

The following arguments are supported:

* `gzip` - (Optional) Specify whether or not to gzip the rendered output.

* `base64_encode` - (Optional) Base64 encoding of the rendered output.

* `part` - (Required) One may specify this many times, this creates a fragment of the rendered cloud-init config file. The order of the parts is maintained in the configuration is maintained in the rendered template.

The `part` block supports:

* `filename` - (Optional) Filename to save part as.

* `content_type` - (Optional) Content type to send file as.

* `content` - (Required) Body for the part.

* `merge_type` - (Optional) Gives the ability to merge multiple blocks of cloud-config together.

## Attributes Reference

The following attributes are exported:

* `rendered` - The final rendered multi-part cloudinit config.
