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

run "test_failed" {
  variables {
    password = run.setup.password
    complex = {
      foo = "bar"
      baz = run.test.password
    }
  }

  assert {
    condition = var.complex == {
      foo = "bar"
      baz = test_resource.resource.id
    }
    error_message = "expected to fail"
  }
}
