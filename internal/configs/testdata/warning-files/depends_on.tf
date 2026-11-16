resource "null_resource" "a" {
}

resource "null_resource" "b" {
  depends_on = ["null_resource.a"] # WARNING: Quoted references are deprecated
}
