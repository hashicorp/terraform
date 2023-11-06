run "setup" {
  variables {
    password = "password"
  }

  module {
    source = "./setup"
  }
}

run "test" {
  variables {
    password = run.setup.password
  }
}
