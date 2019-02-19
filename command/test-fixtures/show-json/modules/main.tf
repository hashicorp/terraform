module "test" {
  source   = "./foo"
  test_var = "baz"
}

output "test" {
  value = module.test.test
}
