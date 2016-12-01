resource "aws_instance" "bar" {
    count = 2
    foo = "bar"
    lifecycle { create_before_destroy = true }
}
