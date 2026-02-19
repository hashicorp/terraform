
module "child" {
  source = "./child"
}

locals {
  result_1 = "${module.child.result}"
  result_2 = "${local.result_1}"
  result_3 = "${local.result_2} world"
}
