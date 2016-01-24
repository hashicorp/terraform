---
layout: "aws"
page_title: "AWS: aws_glacier_vault"
sidebar_current: "docs-aws-resource-glacier-vault"
description: |-
  Provides a Glacier Vault.
---

# aws\_glacier\_vault

Provides a Glacier Vault Resource. You can refer to the [Glacier Developer Guide](https://docs.aws.amazon.com/amazonglacier/latest/dev/working-with-vaults.html) for a full explanation of the Glacier Vault functionality

~> **NOTE:** When removing a Glacier Vault, the Vault must be empty.

## Example Usage

```
resource "aws_sns_topic" "aws_sns_topic" {
  name = "glacier-sns-topic"
}

resource "aws_glacier_vault" "my_archive" {
    name = "MyArchive"

    notification {
      sns_topic = "${aws_sns_topic.aws_sns_topic.arn}"
      events = ["ArchiveRetrievalCompleted","InventoryRetrievalCompleted"]
    }

    access_policy = <<EOF
{
    "Version":"2012-10-17",
    "Statement":[
       {
          "Sid": "add-read-only-perm",
          "Principal": "*",
          "Effect": "Allow",
          "Action": [
             "glacier:InitiateJob",
             "glacier:GetJobOutput"
          ],
          "Resource": "arn:aws:glacier:eu-west-1:432981146916:vaults/MyArchive"
       }
    ]
}
EOF

    tags {
      Test = "MyArchive"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Vault. Names can be between 1 and 255 characters long and the valid characters are a-z, A-Z, 0-9, '\_' (underscore), '-' (hyphen), and '.' (period).
* `access_policy` - (Optional) The policy document. This is a JSON formatted string.
  The heredoc syntax or `file` function is helpful here. Use the [Glacier Developer Guide](https://docs.aws.amazon.com/amazonglacier/latest/dev/vault-access-policy.html) for more information on Glacier Vault Policy
* `notification` - (Optional) The notifications for the Vault. Fields documented below.
* `tags` - (Optional) A mapping of tags to assign to the resource.

**notification** supports the following:

* `events` - (Required) You can configure a vault to publish a notification for `ArchiveRetrievalCompleted` and `InventoryRetrievalCompleted` events.
* `sns_topic` - (Required) The SNS Topic ARN.

## Attributes Reference

The following attributes are exported:

* `location` - The URI of the vault that was created.
* `arn` - The ARN of the vault.
