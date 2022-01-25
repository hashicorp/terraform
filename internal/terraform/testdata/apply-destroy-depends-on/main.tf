resource "aws_instance" "foo" {
    depends_on = ["aws_instance.bar"]
}

resource "aws_instance" "bar" {}
