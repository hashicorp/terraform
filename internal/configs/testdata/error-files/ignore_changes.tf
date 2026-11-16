resource "null_resource" "all" {
  lifecycle {
    ignore_changes = ["*"] # ERROR: Invalid ignore_changes wildcard
  }
}
