mock_resource "aws_s3_bucket" {}

override_resource {
  target = aws_s3_bucket.my_bucket
}
