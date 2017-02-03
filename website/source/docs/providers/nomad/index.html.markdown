---
layout: "nomad"
page_title: "Provider: Nomad"
sidebar_current: "docs-nomad-index"
description: |-
  Nomad is a cluster scheduler. The Nomad provider exposes resources to interact with a Nomad cluster.
---

# Nomad Provider

[Nomad](https://www.nomadproject.io) is a cluster scheduler. The Nomad
provider exposes resources to interact with a Nomad cluster.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Nomad provider
provider "nomad" {
    address = "nomad.mycompany.com"
    region = "us-east-2"
}

# Register a job
resource "nomad_job" "monitoring" {
    jobspec = "${file("${path.module}/jobspec.hcl")}"
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Optional) The HTTP(S) API address of the Nomad agent to use. Defaults to `http://127.0.0.1:4646`. The `NOMAD_ADDR` environment variable can also be used.
* `region` - (Optional) The Nomad region to target. The `NOMAD_REGION` environment variable can also be used.
