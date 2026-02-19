# If read, this file should cause issues. But, it should be ignored.

mock_resource "aws_s3_bucket" {}

mock_data "aws_s3_bucket" {}

override_resource {
  target = aws_s3_bucket.my_bucket
}
