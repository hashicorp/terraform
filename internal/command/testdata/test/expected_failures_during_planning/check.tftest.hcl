
run "check_passes" {

  variables {
    input = "abd"
  }

  # Checks are a little different, as they only produce warnings. So in this
  # case we actually expect the whole run block to be fine. It'll produce
  # warnings during the plan, still execute the apply operation, and then
  # validate the check block failed during the apply stage.
  expect_failures = [
    check.cchar,
  ]

}
