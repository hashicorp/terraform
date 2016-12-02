---
layout: "clevercloud"
page_title: "Provider: Clever Cloud"
sidebar_current: "docs-clevercloud-index"
description: |-
  Clever Cloud is a PaaS provider, no infrastructure management,
  no scaling headache, just $ git push. The Clever Cloud provider 
  exposes resources used to interact with your Clever Cloud 
  organisation. Configuration of the provider is required, as it 
  needs OAuth2 credentials.
---

# Clever Cloud Provider

[Clever Cloud](https://www.clever-cloud.com/) is a PaaS provider, no infrastructure management, 
no scaling headache, just $ git push. The Clever Cloud provider exposes resources used to interact 
with your Clever Cloud organisation. Configuration of the provider is required, as it needs 
OAuth2 credentials.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Clever Cloud provider
provider "clevercloud" {
    org_id = "orga_########-####-####-####-############"
    token = "${var.clevercloud_org_token}"
    secret = "${var.clevercloud_org_secret}"
}

# Minimal configuration to setup a NodeJS application with a PostgreSQL database in Clever Cloud
resource "clevercloud_application_node" "helloworld" {
    name = "Hello World API"
    environment = "${clevercloud_addon_postgresql.helloworld.environment}"
}

resource "clevercloud_addon_postgresql" "helloworld" {
    name = "Hello World PostgreSQL"
    plan = "dev"
}

output "git_remote" { value = "${clevercloud_application_node.helloworld.git_ssh}" }
```

## Argument Reference

The following arguments are supported:

* `org_id` - (Required) The id of organisation to use.
* `token` - (Required) Your personal connection token. You can generate a new one [here](https://console.clever-cloud.com/cli-oauth).
* `secret` - (Required) Your personal connection secret. You can generate a new one [here](https://console.clever-cloud.com/cli-oauth).
* `endpoint`- (Optional) A path Clever Cloud API, required if you use Clever Cloud Enterprise on your private cloud. Default to `https://api.clever-cloud.com/v2/`.
