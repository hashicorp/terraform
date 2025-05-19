resource "aws_instance" "test" {
  provider = aws
  count = 1
  tags = {
    Name = "test"
  }
  
}