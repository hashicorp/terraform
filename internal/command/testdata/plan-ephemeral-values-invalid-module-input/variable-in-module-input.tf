variable "test-eph" {
  type      = string
  default   = "foo"
  ephemeral = true
}

module "test" {
  source = "./eph-module"
  eph    = var.test-eph
}
