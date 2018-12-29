variable "foo" {}

resource "aws_instance" "web" {
    count = "nope"
}
