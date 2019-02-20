module "module_test" {
  source   = "./foo"
  test_var = "baz"
}

output "test" {
  value      = module.module_test.test
  depends_on = [module.module_test]
}
