resource "test_object" "A" {}

resource "test_object" "B" {
  count       = 2
  test_string = "${test_object.A.*.test_string}"
}
