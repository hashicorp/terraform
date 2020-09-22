resource "aws_instance" "web" {
    count = 3
}
