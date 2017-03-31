---
layout: "elasticsearch"
page_title: "Provider: Elasticsearch"
sidebar_current: "docs-elasticsearch-index"
description: |-
  The Elasticsearch provider is used to interact with the resources supported by Elasticsearch. The provider needs to be configured with an endpoint URL before it can be used.
---

# Elasticsearch Provider

The Elasticsearch provider is used to interact with the 
resources supported by Elasticsearch. The provider needs 
to be configured with an endpoint URL before it can be used.

AWS Elasticsearch Service domains are supported.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Elasticsearch provider
provider "elasticsearch" {
  url = "http://127.0.0.1:9200"
}

# Create an index template
resource "elasticsearch_index_template" "template_1" {
  name = "template_1"
  body = <<EOF
{
  "template": "te*",
  "settings": {
    "number_of_shards": 1
  },
  "mappings": {
    "type1": {
      "_source": {
        "enabled": false
      },
      "properties": {
        "host_name": {
          "type": "keyword"
        },
        "created_at": {
          "type": "date",
          "format": "EEE MMM dd HH:mm:ss Z YYYY"
        }
      }
    }
  }
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `url` - (Required) The Elasticsearch endpoint URL. It must be provided, but it can also be sourced from the `ELASTICSEARCH_URL` environment variable.
* `aws_access_key` - (Optional) The access key for use with AWS Elasticsearch Service domains. It can also be sourced from the `AWS_ACCESS_KEY_ID` environment variable.
* `aws_secret_key` - (Optional) The secret key for use with AWS Elasticsearch Service domains. It can also be sourced from the `AWS_SECRET_ACCESS_KEY` environment variable.
* `aws_token` - (Optional) The session token for use with AWS Elasticsearch Service domains. It can also be sourced from the `AWS_SESSION_TOKEN` environment variable.
