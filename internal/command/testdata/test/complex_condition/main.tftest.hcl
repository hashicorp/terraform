
variables {
  foo = [{
    bar = "baz",
    qux = "quux",
  }]
}

run "validate_output" {
  assert {
    condition = output.foo == var.foo[0]
    error_message = "expected to fail"
  }
}

run "validate_complex_output" {
  assert {
    condition = output.complex == var.foo
    error_message = "expected to fail"
  }
}

run "validate_complex_output_pass" {
  assert {
    condition = output.complex != var.foo
    error_message = "should pass"
  }
}
