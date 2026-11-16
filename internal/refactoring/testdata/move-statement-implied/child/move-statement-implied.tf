# This fixture is useful only in conjunction with a previous run state that
# conforms to the statements encoded in the resource names. It's for
# TestImpliedMoveStatements only.

resource "foo" "formerly_count" {
  # but not count anymore
}

resource "foo" "now_count" {
  count = 1
}

moved {
  from = foo.no_longer_present[1]
  to   = foo.no_longer_present
}
