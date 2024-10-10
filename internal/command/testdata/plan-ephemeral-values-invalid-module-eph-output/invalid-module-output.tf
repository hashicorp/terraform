module "test" {
  source = "./eph-module"
  eph    = "foo"
}

output "eph" {
  ephemeral = true
  value     = module.test.not-eph
}
