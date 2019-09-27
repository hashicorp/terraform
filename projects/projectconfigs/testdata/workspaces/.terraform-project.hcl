workspace "local" {
  for_each = {}

  config    = "./foo"
  variables = {}

  state_storage "local" {
  }
}

workspace "remote" {
  remote    = "tf.example.com/foo/bar"
  config    = "./foo"
  variables = {}
}
