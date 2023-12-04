
run "test" {}

run "verify" {
  module {
    source = "./verify"
  }

  variables {
    id = run.test.id
  }
}

run "test_two" {}