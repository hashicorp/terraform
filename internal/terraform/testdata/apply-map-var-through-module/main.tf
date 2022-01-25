variable "amis_in" {
  type = map(string)
  default = {
    "us-west-1" = "ami-123456"
    "us-west-2" = "ami-456789"
    "eu-west-1" = "ami-789012"
    "eu-west-2" = "ami-989484"
  }
}

module "test" {
  source = "./amodule"

  amis = var.amis_in
}

output "amis_from_module" {
  value = module.test.amis_out
}
