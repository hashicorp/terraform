resource "test_object" "a" {}

resource "test_object" "b" {
  test_string = "${test_object.a.test_string}"
}

resource "test_object" "c" {
  test_string = "${test_object.b.test_string}"
}
