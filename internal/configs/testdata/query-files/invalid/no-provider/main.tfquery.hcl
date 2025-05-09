list "aws_instance" "test" {
  count = 1
  tags = {
    Name = "test"
  }
}