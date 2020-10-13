variable "module_version" { default = "v1.0" }

module "foo" {
  source  = "./ff"
  version = var.module_version
}
