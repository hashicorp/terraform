resource "test_object" "create" {
  provisioner "test" {}
}

resource "test_object" "other" {
  test_string = "${test_object.create.test_string}"
}
