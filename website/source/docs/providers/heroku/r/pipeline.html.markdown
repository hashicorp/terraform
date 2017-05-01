---
layout: "heroku"
page_title: "Heroku: heroku_pipeline_"
sidebar_current: "docs-heroku-resource-pipeline-x"
description: |-
  Provides a Heroku Pipeline resource.
---

# heroku\_pipeline


Provides a [Heroku Pipeline](https://devcenter.heroku.com/articles/pipelines)
resource.

A pipeline is a group of Heroku apps that share the same codebase. Once a
pipeline is created, and apps are added to different stages using
[`heroku_pipeline_coupling`](./pipeline_coupling.html), you can promote app
slugs to the next stage.

## Example Usage

```hcl
# Create Heroku apps for staging and production
resource "heroku_app" "staging" {
  name = "test-app-staging"
}

resource "heroku_app" "production" {
  name = "test-app-production"
}

# Create a Heroku pipeline
resource "heroku_pipeline" "test-app" {
  name = "test-app"
}

# Couple apps to different pipeline stages
resource "heroku_pipeline_coupling" "staging" {
  app      = "${heroku_app.staging.name}"
  pipeline = "${heroku_pipeline.test-app.id}"
  stage    = "staging"
}

resource "heroku_pipeline_coupling" "production" {
  app      = "${heroku_app.production.name}"
  pipeline = "${heroku_pipeline.test-app.id}"
  stage    = "production"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the pipeline.

## Attributes Reference

The following attributes are exported:

* `id` - The UUID of the pipeline.
* `name` - The name of the pipeline.
