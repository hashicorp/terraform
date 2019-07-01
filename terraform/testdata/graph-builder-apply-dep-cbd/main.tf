resource "test_object" "A" {
  lifecycle {
    create_before_destroy = true
  }
}

resource "test_object" "B" {
  test_list = test_object.A.*.test_string
}
