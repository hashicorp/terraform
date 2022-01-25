resource "test_object" "A" {
  count = 2
  lifecycle {
    create_before_destroy = true
  }
}

resource "test_object" "B" {
  count       = 2
  test_string = test_object.A[*].test_string[count.index]
}
