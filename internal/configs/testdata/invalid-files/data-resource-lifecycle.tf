data "example" "example" {
  lifecycle {
    # The lifecycle arguments are not valid for data resources:
    # only the precondition and postcondition blocks are allowed.
    ignore_changes = []
  }
}
