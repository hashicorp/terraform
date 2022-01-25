module "module_test_foo" {
  source   = "./foo"
  test_var = "baz"
}

module "module_test_bar" {
  source = "./bar"
}

output "test" {
  value      = module.module_test_foo.test
  depends_on = [module.module_test_foo]
}
