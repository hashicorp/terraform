provider "aws" {
  alias = "prod"

  region = "us-east-1"
  access_key = "${var.prod_access_key}"
  secret_key = "${var.prod_secret_key}"
}

resource "aws_s3_bucket" "prod" {
  provider = "aws.prod"

  bucket = "${var.bucket_name}"
  acl = "private"
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowTest",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::${var.test_account_id}:root"
      },
      "Action": "s3:*",
      "Resource": "arn:aws:s3:::${var.bucket_name}/*"
    }
  ]
}
POLICY
}

resource "aws_s3_bucket_object" "prod" {
  provider = "aws.prod"

  bucket = "${aws_s3_bucket.prod.id}"
  key = "object-uploaded-via-prod-creds"
  source = "${path.module}/prod.txt"
}

provider "aws" {
  alias = "test"

  region = "us-east-1"
  access_key = "${var.test_access_key}"
  secret_key = "${var.test_secret_key}"
}

resource "aws_s3_bucket_object" "test" {
  provider = "aws.test"

  bucket = "${aws_s3_bucket.prod.id}"
  key = "object-uploaded-via-test-creds"
  source = "${path.module}/test.txt"
}
