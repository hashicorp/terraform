resource "test_instance" "a" {
  foo = "a"
}

resource "test_instance" "b" {
  foo = "b"
  lifecycle {
    create_before_destroy = true
  }
}

resource "test_instance" "c" {
  foo = "c"
  lifecycle {
    create_before_destroy = true
  }
}
