resource "test_thing" "zero" {
  count = 0
}

resource "test_thing" "one" {
  count = 1
}
