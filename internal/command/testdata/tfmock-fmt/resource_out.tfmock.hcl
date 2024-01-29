mock_resource "aws_s3_bucket" {
  defaults = {
  arn = "arn:aws:s3:::name" }
}

override_resource {
  target = aws_launch_template.vm
  values = { id = "lt-xyz" }
}
