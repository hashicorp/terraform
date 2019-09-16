resource "aws_instance" "web" {
    ami = "foo"
    lifecycle {
        prevent_destroy = "true"
    }
}

resource "aws_instance" "bar" {
    ami = "foo"
}
