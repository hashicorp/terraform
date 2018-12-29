resource "test_instance" "foo" {
  count = 5
}

resource "test_instance" "bar" {
  # This is invalid because timestamp() returns an unknown value during plan,
  # but the "count" argument in particular must always be known during plan
  # so we can predict how many instances we will operate on.
  count = timestamp()
}
