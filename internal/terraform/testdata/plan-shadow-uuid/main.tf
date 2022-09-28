resource "aws_instance" "test" {
    value = "${uuid()}"
}
