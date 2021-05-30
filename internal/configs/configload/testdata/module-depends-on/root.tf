module "child" {
  source = "./child"
  depends_on = [test_resource.a]
}

resource "test_resource" "a" {
}
