---
layout: "arukas"
page_title: "Provider: Arukas"
sidebar_current: "docs-arukas-index"
description: |-
  The Arukas provider is used to interact with the resources supported by Arukas.
---

# Arukas Provider

The Arukas provider is used to manage [Arukas](https://arukas.io/en/) resources.

Use the navigation to the left to read about the available resources.

For additional details please refer to [Arukas documentation](https://arukas.io/en/category/documents-en/).

## Example Usage

Here is an example that will setup the following:

+ A container resource using the "NGINX" image
+ Instance count is 1
+ Memory size is 256Mbyte
+ Expose tcp 80 port to the EndPoint
+ Set environments variable with like "key1=value1"

Add the below to a file called `arukas.tf` and run the `terraform` command from the same directory:

```hcl
provider "arukas" {
  token  = ""
  secret = ""
}

resource "arukas_container" "foobar" {
  name      = "terraform_for_arukas_test_foobar"
  image     = "nginx:latest"
  instances = 1
  memory    = 256

  ports = {
    protocol = "tcp"
    number   = "80"
  }

  environments {
    key   = "key1"
    value = "value1"
  }
}
```

You'll need to provide your Arukas API token and secret,
so that Terraform can connect. If you don't want to put
credentials in your configuration file, you can leave them
out:

```hcl
provider "arukas" {}
```

...and instead set these environment variables:

- `ARUKAS_JSON_API_TOKEN` : Your Arukas API token
- `ARUKAS_JSON_API_SECRET`: Your Arukas API secret

## Argument Reference

The following arguments are supported:

* `token` - (Required) This is the Arukas API token. It must be provided, but
  it can also be sourced from the `ARUKAS_JSON_API_TOKEN` environment variable.

* `secret` - (Required) This is the Arukas API secret. It must be provided, but
  it can also be sourced from the `ARUKAS_JSON_API_SECRET` environment variable.

* `api_url` - (Optional) Override Arukas API Root URL. Also taken from the `ARUKAS_JSON_API_URL`
  environment variable if provided.

* `trace` - (Optional) The flag of Arukas API trace log. Also taken from the `ARUKAS_DEBUG`
  environment variable if provided.

* `timeout` - (Optional) Override Arukas API timeout seconds. Also taken from the `ARUKAS_TIMEOUT`
  environment variable if provided.
