resource "test_object" "A" {
  count = 1
}

resource "test_object" "B" {
  test_list = test_object.A.*.test_string
}
