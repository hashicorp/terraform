module "mod" {
  source = "./mod"
}

output "out" {
  value = module.mod.base_config["base_template"]
}

resource "template_instance" "root_template" {
  foo = module.mod.base_config["base_template"]
}
