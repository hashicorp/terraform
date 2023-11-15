import {
  id = "foo/bar"
  to = local_file.foo_bar
}

moved {
  from = local_file.foo_bar
  to   = local_file.bar_baz
}

resource "local_file" "bar_baz" {
}
