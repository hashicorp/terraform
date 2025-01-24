resource "test_thing" "zero" {
  count = 0
}

resource "test_thing" "one" {
  count = 2

  lifecycle {
    concurrency = 1
  }
}
