list "aws_instance" "test" {
  provider = aws
  count = 1
  config {
    tags = {
      Name = "test"
    }
  }
}
