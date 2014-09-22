
resource "aws_instance" "web" {
    ami = "foo"
    create_before_destroy = true
}

resource "aws_instance" "bar" {
    ami = "foo"
    create_before_destroy = false
}
