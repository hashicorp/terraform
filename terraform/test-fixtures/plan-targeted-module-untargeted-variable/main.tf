resource "aws_instance" "blue" { }
resource "aws_instance" "green" { }

module "blue_mod" {
  source = "./child"
  id = "${aws_instance.blue.id}"
}

module "green_mod" {
  source = "./child"
  id = "${aws_instance.green.id}"
}
