
variables {
  foo = [{
    bar = "baz",
    qux = "quux",
  }]
}

run "validate_diff_types" {
  variables {
    tr1 = {
    "iops" = tonumber(null)
    "size" = 60
}
    tr2 = {
    iops = null
    size = 60
}
  }
  assert {
    condition = var.tr1 == var.tr2 
    error_message = "expected to fail"
  }
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

run "validate_complex_output_sensitive" {
  assert {
    condition = output.complex == output.complex_sensitive
    error_message = "expected to fail"
  }
}

run "validate_complex_output_pass" {
  assert {
    condition = output.complex != var.foo
    error_message = "should pass"
  }
}
