variable "mod_count_child" { }

module "child2" {
  source    = "./child2"
  mod_count_child2 = "${var.mod_count_child}"
}

resource "aws_instance" "foo" { }
