run "foo" {
    module {
        source = "./fixtures"
    }
    assert {
      condition = output.name == true
      error_message = "foo"
    }
}