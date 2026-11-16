resource "test_thing" "a" {
}

resource "test_thing" "b" {
}

resource "test_thing" "c" {
  dynamic "nested" {
    for_each = test_thing.a.list
    content {
      foo = test_thing.b.id
    }
  }
}
