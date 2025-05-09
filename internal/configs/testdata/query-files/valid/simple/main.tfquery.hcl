list "aws_instance" "test" {
  provider = aws
  count = 1
  tags = {
    Name = "test"
  }
}
list "aws_instance" "test2" {
  provider = aws
  count = 1
  tags = {
    Name = "test2"
  }
}