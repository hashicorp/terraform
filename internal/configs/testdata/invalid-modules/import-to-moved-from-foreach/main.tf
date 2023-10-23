import {
  id = "foo/bar"
  to = local_file.foo_bar["a"]
}

moved {
  from = local_file.foo_bar["a"]
  to   = local_file.foo_baz
}

resource "local_file" "foo_baz" {
  for_each = ["a", "b"]
}
