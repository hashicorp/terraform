provider "aws" {}
resource "aws_instance" "db" {}
resource "aws_instance" "web" {
    foo = "${aws_instance.db.id}"
}
