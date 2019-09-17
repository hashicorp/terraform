module "a_module" {
  source = "./a"
}

module "b_module" {
  source = "./b"
  a_id = "${module.a_module.a_output}"
}
