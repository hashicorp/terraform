---
layout: "elasticsearch"
page_title: "Elasticsearch: elasticsearch_snapshot_repository"
sidebar_current: "docs-elasticsearch-resource-snapshot-repository"
description: |-
  Provides an Elasticsearch snapshot repository resource.
---

# elasticsearch\_snapshot\_repository

Provides an Elasticsearch snapshot repository resource.

## Example Usage

```
# Create a snapshot repository
resource "elasticsearch_snapshot_repository" "repo" {
  name = "es-index-backups"
  type = "s3"
  settings {
    bucket = "es-index-backups"
    region = "us-east-1"
    role_arn = "arn:aws:iam::123456789012:role/MyElasticsearchRole"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the repository.
* `type` - (Required) The name of the repository backend (required plugins must be installed).
* `settings` - (Optional) The settings map applicable for the backend (documented [here](https://www.elastic.co/guide/en/elasticsearch/reference/current/modules-snapshots.html) for official plugins).

## Attributes Reference

The following attributes are exported:

* `id` - The name of the snapshot repository.

