provider "aws" {
    from = "child"
    to = "child"
}

resource "aws_instance" "foo" {
    from = "child"
}
