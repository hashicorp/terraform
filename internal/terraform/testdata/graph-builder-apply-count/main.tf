resource "test_object" "A" {
  count = 2
}

resource "test_object" "B" {
  test_list = test_object.A.*.test_string
}
