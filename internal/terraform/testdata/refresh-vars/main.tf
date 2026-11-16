resource "aws_instance" "web" {}

resource "aws_instance" "db" {
  ami = "${aws_instance.web.id}"
}
