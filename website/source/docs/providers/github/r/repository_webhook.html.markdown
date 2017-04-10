---
layout: "github"
page_title: "GitHub: github_repository_webhook"
sidebar_current: "docs-github-resource-repository-webhook"
description: |-
  Creates and manages repository webhooks within Github organizations
---

# github_repository_webhook

This resource allows you to create and manage webhooks for repositories within your
Github organization.

This resource cannot currently be used to manage webhooks for *personal* repositories,
outside of organizations.

## Example Usage

```hcl
resource "github_repository" "repo" {
  name         = "foo"
  description  = "Terraform acceptance tests"
  homepage_url = "http://example.com/"

  private = false
}

resource "github_repository_webhook" "foo" {
  repository = "${github_repository.repo.name}"

  name = "web"

  configuration {
    url          = "https://google.de/"
    content_type = "form"
    insecure_ssl = false
  }

  active = false

  events = ["issues"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The type of the webhook. See a list of [available hooks](https://api.github.com/hooks).

* `repository` - (Required) The repository of the webhook.

* `events` - (Required) A list of events which should trigger the webhook. Defaults to `["push"]`. See a list of [available events](https://developer.github.com/v3/activity/events/types/)

* `config` - (Required) key/value pair of configuration for this webhook. Available keys are `url`, `content_type`, `secret` and `insecure_ssl`.

* `active` - (Optional) Indicate of the webhook should receive events. Defaults to `true`.

## Attributes Reference

The following additional attributes are exported:

* `url` - URL of the webhook
