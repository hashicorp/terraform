variable "mod_count_child2" { }

resource "aws_instance" "foo" {
  count = "${var.mod_count_child2}"
}
