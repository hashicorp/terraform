resource "test_object" "A" {
  lifecycle {
    create_before_destroy = true
  }
}

resource "test_object" "B" {
  lifecycle {
    create_before_destroy = true
  }
}

resource "test_object" "C" {
  test_string = "${test_object.A.id}-${test_object.B.id}"
}
