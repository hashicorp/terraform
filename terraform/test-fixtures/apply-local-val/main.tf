
module "child" {
  source = "./child"
}

locals {
  result_1 = "${module.child.result}"
  result_2 = "${local.result_1}"
  result_3 = "${local.result_2} world"
}

output "result_1" {
  value = "${local.result_1}"
}

output "result_3" {
  value = "${local.result_3}"
}
