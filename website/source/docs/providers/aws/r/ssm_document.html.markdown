---
layout: "aws"
page_title: "AWS: aws_ssm_document"
sidebar_current: "docs-aws-resource-ssm-document"
description: |-
  Provides an SSM Document resource
---

# aws\_ssm\_document

Provides an SSM Document resource

~> **NOTE on updating SSM documents:** Only documents with a schema version of 2.0
or greater can update their content once created, see [SSM Schema Features][1]. To update a document with an older
schema version you must recreate the resource.

## Example Usage

```hcl
resource "aws_ssm_document" "foo" {
  name          = "test_document"
  document_type = "Command"

  content = <<DOC
  {
    "schemaVersion": "1.2",
    "description": "Check ip configuration of a Linux instance.",
    "parameters": {

    },
    "runtimeConfig": {
      "aws:runShellScript": {
        "properties": [
          {
            "id": "0.aws:runShellScript",
            "runCommand": ["ifconfig"]
          }
        ]
      }
    }
  }
DOC
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the document.
* `content` - (Required) The json content of the document.
* `document_type` - (Required) The type of the document. Valid document types include: `Command`, `Policy` and `Automation`
* `permissions` - (Optional) Additional Permissions to attach to the document. See [Permissions](#permissions) below for details.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the document.
* `content` -  The json content of the document.
* `created_date` - The date the document was created.
* `description` - The description of the document.
* `schema_version` - The schema version of the document.
* `document_type` - The type of document created.
* `default_version` - The default version of the document.
* `hash` - The sha1 or sha256 of the document content
* `hash_type` - "Sha1" "Sha256". The hashing algorithm used when hashing the content.
* `latest_version` - The latest version of the document.
* `owner` - The AWS user account of the person who created the document.
* `status` - "Creating", "Active" or "Deleting". The current status of the document.
* `parameter` - The parameters that are available to this document.
* `permissions` - The permissions of how this document should be shared.
* `platform_types` - A list of OS platforms compatible with this SSM document, either "Windows" or "Linux".

[1]: http://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-ssm-docs.html#document-schemas-features

## Permissions

The permissions attribute specifies how you want to share the document. If you share a document privately,
you must specify the AWS user account IDs for those people who can use the document. If you share a document
publicly, you must specify All as the account ID.

The permissions mapping supports the following:

* `type` - The permission type for the document. The permission type can be `Share`.
* `account_ids` - The AWS user accounts that should have access to the document. The account IDs can either be a group of account IDs or `All`.
