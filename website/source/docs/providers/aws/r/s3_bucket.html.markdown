---
layout: "aws"
page_title: "AWS: aws_s3_bucket"
sidebar_current: "docs-aws-resource-s3-bucket"
description: |-
  Provides a S3 bucket resource.
---

# aws\_s3\_bucket

Provides a S3 bucket resource.

## Example Usage

### Private Bucket w/ Tags

```hcl
resource "aws_s3_bucket" "b" {
  bucket = "my_tf_test_bucket"
  acl    = "private"

  tags {
    Name        = "My bucket"
    Environment = "Dev"
  }
}
```

### Static Website Hosting

```hcl
resource "aws_s3_bucket" "b" {
  bucket = "s3-website-test.hashicorp.com"
  acl    = "public-read"
  policy = "${file("policy.json")}"

  website {
    index_document = "index.html"
    error_document = "error.html"

    routing_rules = <<EOF
[{
    "Condition": {
        "KeyPrefixEquals": "docs/"
    },
    "Redirect": {
        "ReplaceKeyPrefixWith": "documents/"
    }
}]
EOF
  }
}
```

### Using CORS

```hcl
resource "aws_s3_bucket" "b" {
  bucket = "s3-website-test.hashicorp.com"
  acl    = "public-read"

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT", "POST"]
    allowed_origins = ["https://s3-website-test.hashicorp.com"]
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
}
```

### Using versioning

```hcl
resource "aws_s3_bucket" "b" {
  bucket = "my_tf_test_bucket"
  acl    = "private"

  versioning {
    enabled = true
  }
}
```

### Enable Logging

```hcl
resource "aws_s3_bucket" "log_bucket" {
  bucket = "my_tf_log_bucket"
  acl    = "log-delivery-write"
}

resource "aws_s3_bucket" "b" {
  bucket = "my_tf_test_bucket"
  acl    = "private"

  logging {
    target_bucket = "${aws_s3_bucket.log_bucket.id}"
    target_prefix = "log/"
  }
}
```

### Using object lifecycle

```hcl
resource "aws_s3_bucket" "bucket" {
  bucket = "my-bucket"
  acl    = "private"

  lifecycle_rule {
    id      = "log"
    prefix  = "log/"
    enabled = true

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 60
      storage_class = "GLACIER"
    }

    expiration {
      days = 90
    }
  }

  lifecycle_rule {
    id      = "tmp"
    prefix  = "tmp/"
    enabled = true

    expiration {
      date = "2016-01-12"
    }
  }
}

resource "aws_s3_bucket" "versioning_bucket" {
  bucket = "my-versioning-bucket"
  acl    = "private"

  versioning {
    enabled = true
  }

  lifecycle_rule {
    prefix  = "config/"
    enabled = true

    noncurrent_version_transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    noncurrent_version_transition {
      days          = 60
      storage_class = "GLACIER"
    }

    noncurrent_version_expiration {
      days = 90
    }
  }
}
```

### Using replication configuration

```hcl
provider "aws" {
  region = "eu-west-1"
}

provider "aws" {
  alias  = "central"
  region = "eu-central-1"
}

resource "aws_iam_role" "replication" {
  name = "tf-iam-role-replication-12345"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "s3.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_policy" "replication" {
  name = "tf-iam-role-policy-replication-12345"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:GetReplicationConfiguration",
        "s3:ListBucket"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.bucket.arn}"
      ]
    },
    {
      "Action": [
        "s3:GetObjectVersion",
        "s3:GetObjectVersionAcl"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.bucket.arn}/*"
      ]
    },
    {
      "Action": [
        "s3:ReplicateObject",
        "s3:ReplicateDelete"
      ],
      "Effect": "Allow",
      "Resource": "${aws_s3_bucket.destination.arn}/*"
    }
  ]
}
POLICY
}

resource "aws_iam_policy_attachment" "replication" {
  name       = "tf-iam-role-attachment-replication-12345"
  roles      = ["${aws_iam_role.replication.name}"]
  policy_arn = "${aws_iam_policy.replication.arn}"
}

resource "aws_s3_bucket" "destination" {
  bucket   = "tf-test-bucket-destination-12345"
  region   = "eu-west-1"

  versioning {
    enabled = true
  }
}

resource "aws_s3_bucket" "bucket" {
  provider = "aws.central"
  bucket   = "tf-test-bucket-12345"
  acl      = "private"
  region   = "eu-central-1"

  versioning {
    enabled = true
  }

  replication_configuration {
    role = "${aws_iam_role.replication.arn}"

    rules {
      id     = "foobar"
      prefix = "foo"
      status = "Enabled"

      destination {
        bucket        = "${aws_s3_bucket.destination.arn}"
        storage_class = "STANDARD"
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Optional, Forces new resource) The name of the bucket. If omitted, Terraform will assign a random, unique name.
* `bucket_prefix` - (Optional, Forces new resource) Creates a unique bucket name beginning with the specified prefix. Conflicts with `bucket`.
* `acl` - (Optional) The [canned ACL](https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl) to apply. Defaults to "private".
* `policy` - (Optional) A valid [bucket policy](https://docs.aws.amazon.com/AmazonS3/latest/dev/example-bucket-policies.html) JSON document. Note that if the policy document is not specific enough (but still valid), Terraform may view the policy as constantly changing in a `terraform plan`. In this case, please make sure you use the verbose/specific version of the policy.

* `tags` - (Optional) A mapping of tags to assign to the bucket.
* `force_destroy` - (Optional, Default:false ) A boolean that indicates all objects should be deleted from the bucket so that the bucket can be destroyed without error. These objects are *not* recoverable.
* `website` - (Optional) A website object (documented below).
* `cors_rule` - (Optional) A rule of [Cross-Origin Resource Sharing](https://docs.aws.amazon.com/AmazonS3/latest/dev/cors.html) (documented below).
* `versioning` - (Optional) A state of [versioning](https://docs.aws.amazon.com/AmazonS3/latest/dev/Versioning.html) (documented below)
* `logging` - (Optional) A settings of [bucket logging](https://docs.aws.amazon.com/AmazonS3/latest/UG/ManagingBucketLogging.html) (documented below).
* `lifecycle_rule` - (Optional) A configuration of [object lifecycle management](http://docs.aws.amazon.com/AmazonS3/latest/dev/object-lifecycle-mgmt.html) (documented below).
* `acceleration_status` - (Optional) Sets the accelerate configuration of an existing bucket. Can be `Enabled` or `Suspended`.
* `region` - (Optional) If specified, the AWS region this bucket should reside in. Otherwise, the region used by the callee.
* `request_payer` - (Optional) Specifies who should bear the cost of Amazon S3 data transfer.
Can be either `BucketOwner` or `Requester`. By default, the owner of the S3 bucket would incur
the costs of any data transfer. See [Requester Pays Buckets](http://docs.aws.amazon.com/AmazonS3/latest/dev/RequesterPaysBuckets.html)
developer guide for more information.
* `replication_configuration` - (Optional) A configuration of [replication configuration](http://docs.aws.amazon.com/AmazonS3/latest/dev/crr.html) (documented below).

~> **NOTE:** You cannot use `acceleration_status` in `cn-north-1` or `us-gov-west-1`

The `website` object supports the following:

* `index_document` - (Required, unless using `redirect_all_requests_to`) Amazon S3 returns this index document when requests are made to the root domain or any of the subfolders.
* `error_document` - (Optional) An absolute path to the document to return in case of a 4XX error.
* `redirect_all_requests_to` - (Optional) A hostname to redirect all website requests for this bucket to. Hostname can optionally be prefixed with a protocol (`http://` or `https://`) to use when redirecting requests. The default is the protocol that is used in the original request.
* `routing_rules` - (Optional) A json array containing [routing rules](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-websiteconfiguration-routingrules.html)
describing redirect behavior and when redirects are applied.

The `CORS` object supports the following:

* `allowed_headers` (Optional) Specifies which headers are allowed.
* `allowed_methods` (Required) Specifies which methods are allowed. Can be `GET`, `PUT`, `POST`, `DELETE` or `HEAD`.
* `allowed_origins` (Required) Specifies which origins are allowed.
* `expose_headers` (Optional) Specifies expose header in the response.
* `max_age_seconds` (Optional) Specifies time in seconds that browser can cache the response for a preflight request.

The `versioning` object supports the following:

* `enabled` - (Optional) Enable versioning. Once you version-enable a bucket, it can never return to an unversioned state. You can, however, suspend versioning on that bucket.
* `mfa_delete` - (Optional) Enable MFA delete for either `Change the versioning state of your bucket` or `Permanently delete an object version`. Default is `false`.

The `logging` object supports the following:

* `target_bucket` - (Required) The name of the bucket that will receive the log objects.
* `target_prefix` - (Optional) To specify a key prefix for log objects.

The `lifecycle_rule` object supports the following:

* `id` - (Optional) Unique identifier for the rule.
* `prefix` - (Required) Object key prefix identifying one or more objects to which the rule applies.
* `enabled` - (Required) Specifies lifecycle rule status.
* `abort_incomplete_multipart_upload_days` (Optional) Specifies the number of days after initiating a multipart upload when the multipart upload must be completed.
* `expiration` - (Optional) Specifies a period in the object's expire (documented below).
* `transition` - (Optional) Specifies a period in the object's transitions (documented below).
* `noncurrent_version_expiration` - (Optional) Specifies when noncurrent object versions expire (documented below).
* `noncurrent_version_transition` - (Optional) Specifies when noncurrent object versions transitions (documented below).

At least one of `expiration`, `transition`, `noncurrent_version_expiration`, `noncurrent_version_transition` must be specified.

The `expiration` object supports the following

* `date` (Optional) Specifies the date after which you want the corresponding action to take effect.
* `days` (Optional) Specifies the number of days after object creation when the specific rule action takes effect.
* `expired_object_delete_marker` (Optional) On a versioned bucket (versioning-enabled or versioning-suspended bucket), you can add this element in the lifecycle configuration to direct Amazon S3 to delete expired object delete markers.

The `transition` object supports the following

* `date` (Optional) Specifies the date after which you want the corresponding action to take effect.
* `days` (Optional) Specifies the number of days after object creation when the specific rule action takes effect.
* `storage_class` (Required) Specifies the Amazon S3 storage class to which you want the object to transition. Can be `STANDARD_IA` or `GLACIER`.

The `noncurrent_version_expiration` object supports the following

* `days` (Required) Specifies the number of days an object is noncurrent object versions expire.

The `noncurrent_version_transition` object supports the following

* `days` (Required) Specifies the number of days an object is noncurrent object versions expire.
* `storage_class` (Required) Specifies the Amazon S3 storage class to which you want the noncurrent versions object to transition. Can be `STANDARD_IA` or `GLACIER`.

The `replication_configuration` object supports the following:

* `role` - (Required) The ARN of the IAM role for Amazon S3 to assume when replicating the objects.
* `rules` - (Required) Specifies the rules managing the replication (documented below).

The `rules` object supports the following:

* `id` - (Optional) Unique identifier for the rule.
* `destination` - (Required) Specifies the destination for the rule (documented below).
* `prefix` - (Required) Object keyname prefix identifying one or more objects to which the rule applies. Set as an empty string to replicate the whole bucket.
* `status` - (Required) The status of the rule. Either `Enabled` or `Disabled`. The rule is ignored if status is not Enabled.

The `destination` object supports the following:

* `bucket` - (Required) The ARN of the S3 bucket where you want Amazon S3 to store replicas of the object identified by the rule.
* `storage_class` - (Optional) The class of storage used to store the object.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the bucket.
* `arn` - The ARN of the bucket. Will be of format `arn:aws:s3:::bucketname`.
* `bucket_domain_name` - The bucket domain name. Will be of format `bucketname.s3.amazonaws.com`.
* `hosted_zone_id` - The [Route 53 Hosted Zone ID](https://docs.aws.amazon.com/general/latest/gr/rande.html#s3_website_region_endpoints) for this bucket's region.
* `region` - The AWS region this bucket resides in.
* `website_endpoint` - The website endpoint, if the bucket is configured with a website. If not, this will be an empty string.
* `website_domain` - The domain of the website endpoint, if the bucket is configured with a website. If not, this will be an empty string. This is used to create Route 53 alias records.

## Import

S3 bucket can be imported using the `bucket`, e.g.

```
$ terraform import aws_s3_bucket.bucket bucket-name
```
