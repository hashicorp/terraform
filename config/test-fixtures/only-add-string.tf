resource "aws_instance" "web" {
    ami = "foo"
    lifecycle {
        only_add = "true"
    }
}

resource "aws_instance" "bar" {
    ami = "foo"
}
