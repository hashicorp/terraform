
run "test" {
  variables {
    input = "Hello, world!"
  }
}

run "verify" {
  module {
    source = "./testing"
  }

  variables {
    id = run.test.id
  }
}
