provider "null" {
  alias = "foo"
}

resource "null_resource" "test" {
  provider = "null.foo" # WARNING: Quoted references are deprecated
}
