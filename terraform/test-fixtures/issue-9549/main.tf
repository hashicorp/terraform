module "mod" {
  source = "./mod"
}

resource "template_file" "root_template" {
  template = "ext: ${module.mod.base_config["base_template"]}"
}
