resource "aws_instance" "bar" {
    require_new = "xyz"
    lifecycle {
        create_before_destroy = true
    }
}
