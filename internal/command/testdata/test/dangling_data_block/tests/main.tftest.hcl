run "test" {
  variables {
    input = "Hello, world!"
  }
}

run "verify" {
  module {
    source = "./testing/verify"
  }

  variables {
    id = run.test.id
  }
}
