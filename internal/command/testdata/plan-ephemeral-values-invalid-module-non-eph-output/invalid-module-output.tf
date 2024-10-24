module "test" {
  source = "./eph-module"
  eph    = "foo"
}

output "eph" {
  value = module.test.eph
}
