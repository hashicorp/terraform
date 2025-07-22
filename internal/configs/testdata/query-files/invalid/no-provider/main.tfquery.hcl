list "aws_instance" "test" {
  count = 1
  config {
    tags = {
      Name = "test"
    }
  }
}