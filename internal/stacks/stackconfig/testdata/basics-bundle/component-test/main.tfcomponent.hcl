component "pet-nulls" {
  source  = "app.staging.terraform.io/component-configurations/pet-nulls"
  version = "0.0.2"

  inputs = {
    instances = var.instances
    prefix    = var.prefix
  }
}
