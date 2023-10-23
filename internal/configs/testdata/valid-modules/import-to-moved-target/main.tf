import {
  id = "foo/bar"
  to = local_file.foo_bar["b"]
}

moved {
  from = local_file.foo_old
  to   = local_file.foo_bar["b"]
}

resource "local_file" "foo_bar" {
  for_each = ["a", "b"]
}
