---
layout: "logentries"
page_title: "Provider: Logentries"
sidebar_current: "docs-logentries-index"
description: |-
  The Logentries provider is used to manage Logentries logs and log sets. Logentries provides live log management and analytics. The provider needs to be configured with a Logentries account key before it can be used.
---

# Logentries Provider

The Logentries provider is used to manage Logentries logs and log sets. Logentries provides live log management and analytics. The provider needs to be configured with a Logentries account key before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Logentries provider
provider "logentries" {
  account_key = "${var.logentries_account_key}"
}

# Create a log set
resource "logentries_logset" "host_logs" {
  name = "${var.server}-logs"
}

# Create a log and add it to the log set
resource "logentries_log" "app_log" {
  logset_id = "${logentries_logset.host_logs.id}"
  name      = "myapp-log"
  source    = "token"
}

# Add the log token to a cloud-config that can be used by an
# application to send logs to Logentries
resource "aws_launch_configuration" "app_launch_config" {
  name_prefix   = "myapp-"
  image_id      = "${var.ami}"
  instance_type = "${var.instance_type}"

  user_data = <<EOF
#cloud-config
write_files:
  - content: |
        #!/bin/bash -l
        export LOGENTRIES_TOKEN=${logentries_log.app_log.token}
        run-my-app.sh
    path: "/etc/sv/my-app/run"
    permissions: 0500
runcmd:
  - ln -s /etc/sv/my-app /etc/service/
EOF

  iam_instance_profile = "${var.instance_profile}"

  lifecycle {
    create_before_destroy = true
  }

  root_block_device {
    volume_type = "gp2"
    volume_size = "100"
  }
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `account_key` - (Required) The Logentries account key. This can also be specified with the `LOGENTRIES_ACCOUNT_KEY` environment variable. See the Logentries [account key documentation](https://logentries.com/doc/accountkey/) for more information.