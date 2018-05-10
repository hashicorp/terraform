resource "test_object" "A" {}

resource "test_object" "B" {
  test_string = "${test_object.A.test_string}"
}

resource "test_object" "C" {
  test_string = "${test_object.B.test_string}"
}
