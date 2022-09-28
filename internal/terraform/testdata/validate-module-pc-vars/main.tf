variable "provider_var" {}

module "child" {
    source = "./child"

    value = var.provider_var
}
