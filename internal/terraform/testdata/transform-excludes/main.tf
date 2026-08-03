resource "aws_vpc" "included" {}
resource "aws_vpc" "also_included" {}

resource "aws_subnet" "included" {
  depends_on = [
    aws_vpc.included,
  ]
}
resource "aws_subnet" "also_included" {}

resource "aws_instance" "included" {}

resource "aws_instance" "excluded_directly" {
  depends_on = [
    aws_subnet.included,
  ]
}
resource "aws_instance" "excluded_by_dep" {
  depends_on = [
    aws_instance.excluded_directly,
  ]
}

