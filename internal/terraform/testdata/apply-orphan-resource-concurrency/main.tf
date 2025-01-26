lock "test" {
  concurrency = 1
}

resource "test_thing" "zero" {
  count = 0
}

resource "test_thing" "one" {
  count = 2

  lifecycle {
    lock = lock.test
  }
}
