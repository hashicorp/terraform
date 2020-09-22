resource "null_resource" "one" {
  lifecycle {
    ignore_changes = ["triggers"] # WARNING: Quoted references are deprecated
  }
}

resource "null_resource" "all" {
  lifecycle {
    ignore_changes = ["*"] # WARNING: Deprecated ignore_changes wildcard
  }
}
