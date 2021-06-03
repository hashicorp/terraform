resource "test_object" "A" {}

resource "test_object" "B" {
  test_string = "${test_object.A.test_string}"
}
