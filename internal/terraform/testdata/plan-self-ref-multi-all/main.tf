resource "aws_instance" "web" {
    foo = aws_instance.web[*].id
    count = 4
}
