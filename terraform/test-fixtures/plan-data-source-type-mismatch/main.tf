data "aws_availability_zones" "azs" {}
resource "aws_instance" "foo" {
  ami = "${data.aws_availability_zones.azs.names}"
}
