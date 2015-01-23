resource "aws_instance" "parent" {
  count = 2
}

module "child" {
  source = "./child"
  things = "${join(",", aws_instance.parent.*.id)}"
}

