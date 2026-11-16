
module "example" {
  source = "./example2"

  kept = "primary kept"
  foo  = "primary foo"

  providers = {
    test = test.foo
  }
  depends_on = [null_resource.test]
}
resource "null_resource" "test" {}
