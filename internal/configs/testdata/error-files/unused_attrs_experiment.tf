resource "a" "b" {
  lifecycle {
    # This is invalid only as long as it remains behind an experiment guard.
    # If we later stabilize this feature, this could become valid syntax.
    unused = [foo]
  }
}
