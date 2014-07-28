resource "aws_instance" "web" {
}

resource "aws_instance" "db" {
  ami = "${aws_instance.loadbalancer.foo}"
}
