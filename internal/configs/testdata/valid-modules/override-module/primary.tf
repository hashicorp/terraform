
module "example" {
  source = "./example2"

  kept = "primary kept"
  foo  = "primary foo"

  providers = {
    test = test.foo
  }
}
