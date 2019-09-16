resource "aws_instance" "web" {
    ami = "foo"
    lifecycle {
        ignore_changes = ["ami"]
    }
}

resource "aws_instance" "bar" {
    ami = "foo"
    lifecycle {
        ignore_changes = []
    }
}

resource "aws_instance" "baz" {
  ami = "foo"
}
