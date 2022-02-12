# This fixture is useful only in conjunction with a previous run state that
# conforms to the statements encoded in the resource names. It's for
# TestImpliedMoveStatements only.

resource "foo" "formerly_count" {
  # but not count anymore
}

resource "foo" "now_count" {
  count = 2
}

resource "foo" "new_no_count" {
}

resource "foo" "new_count" {
  count = 2
}

resource "foo" "formerly_count_explicit" {
  # but not count anymore
}

moved {
  from = foo.formerly_count_explicit[1]
  to   = foo.formerly_count_explicit
}

resource "foo" "now_count_explicit" {
  count = 2
}

moved {
  from = foo.now_count_explicit
  to   = foo.now_count_explicit[1]
}

resource "foo" "now_for_each_formerly_count" {
  for_each = { a = 1 }
}

resource "foo" "now_for_each_formerly_no_count" {
  for_each = { a = 1 }
}

resource "foo" "ambiguous" {
  # this one doesn't have count in the config, but the test should
  # set it up to have both no-key and zero-key instances in the
  # state.
}

module "child" {
  source = "./child"
}
