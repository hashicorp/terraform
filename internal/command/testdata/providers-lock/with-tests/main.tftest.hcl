provider "test" {
  alias = "runner"
}

run "test_run" {
  providers = {
    test = test.runner
  }
}
