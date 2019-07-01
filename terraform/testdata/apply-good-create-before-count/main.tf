resource "aws_instance" "bar" {
    count = 2
    require_new = "xyz"
    lifecycle {
        create_before_destroy = true
    }
}
