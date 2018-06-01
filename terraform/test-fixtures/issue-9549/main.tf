module "mod" {
  source = "./mod"
}

resource "template_instance" "root_template" {
  compute_value = "ext: ${module.mod.base_config["base_template"]}"
  compute  = "value"
}
