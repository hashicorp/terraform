import {}

moved {
  from = "local_file.foo"
  to   = "local_file.bar"
}

resource "local_file" "bar" {}
