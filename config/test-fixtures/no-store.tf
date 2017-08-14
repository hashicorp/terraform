resource "aws_instance" "web" {
    ami = "foo"
    lifecycle {
        no_store = ["ami"]
    }
}

resource "aws_instance" "bar" {
    ami = "foo"
    lifecycle {
        no_store = []
    }
}

resource "aws_instance" "baz" {
  ami = "foo"
}
