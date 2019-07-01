module "child" {
  source = "./child"
}

resource "test_object" "create" {}

resource "test_object" "other" {
  test_string = "${test_object.create.test_string}"
}
