module "a" {
  source = "./a"
}

module "b" {
  source = "./b"
  input = "${module.a.output}"
}
