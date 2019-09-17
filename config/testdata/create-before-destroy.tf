
resource "aws_instance" "web" {
    ami = "foo"
    lifecycle {
        create_before_destroy = true
    }
}

resource "aws_instance" "bar" {
    ami = "foo"
    lifecycle {
        create_before_destroy = false
    }
}
