resource "aws_instance" "foo" {
  count         = 2
  ami           = "ami-bcd456"
  lifecycle {
    no_store = ["ami"]
  }
}

resource "aws_eip" "foo" {
  count    = 2
  instance = "${element(aws_instance.foo.*.id, count.index)}"
}
