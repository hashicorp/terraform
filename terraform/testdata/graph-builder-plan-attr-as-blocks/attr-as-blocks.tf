resource "test_thing" "a" {
}

resource "test_thing" "b" {
  nested {
    foo = test_thing.a.id
  }
}
