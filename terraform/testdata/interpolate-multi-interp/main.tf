resource "aws_instance" "web" {
    count = "${var.c}"
}
