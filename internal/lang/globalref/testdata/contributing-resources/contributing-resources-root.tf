variable "environment" {
  type = string
}

data "test_thing" "environment" {
  string = var.environment
}

module "network" {
  source = "./network"

  base_cidr_block = data.test_thing.environment.any.base_cidr_block
  subnet_count    = data.test_thing.environment.any.subnet_count
}

module "compute" {
  source = "./compute"

  network = module.network
}

output "network" {
  value = module.network
}

output "c10s_url" {
  value = module.compute.compuneetees_api_url
}
