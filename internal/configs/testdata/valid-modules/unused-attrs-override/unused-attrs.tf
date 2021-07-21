terraform {
  experiments = [unused_attrs]
}

resource "a" "b" {
  lifecycle {
    unused = [
      foo,
      bar[0],
      baz.boop,
      bleep["bloop"].blah,
    ]
  }
}
