provider "do" {
  foo = "${aws_instance.foo.num}"
}

resource "aws_instance" "foo" {
    num = "2"
}

resource "do_droplet" "bar" {}
