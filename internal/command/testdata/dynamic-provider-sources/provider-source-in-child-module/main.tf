variable "provider_src" {
  type  = string
  const = true
}

module "child" {
  source       = "./modules/child"
  provider_src = var.provider_src
}
