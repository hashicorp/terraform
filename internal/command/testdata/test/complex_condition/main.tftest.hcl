
variables {

  foo = {
    bar = "baz",
    qux = "qux",
    matches = "matches",
    xuq = "nope"
  }

  bar = {
    root = [{
      bar = [1]
      qux = "qux"
      },
      {
        bar = [2]
        qux = "quux"
    }]
  }
}

run "validate_diff_types" {
// the compared values are of different types, but have the same
// visual representation in the terminal.
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
    condition = output.foo == var.foo
    error_message = "expected to fail due to different values"
  }
}

run "validate_complex_output" {
  assert {
    // just a more complex value comparison
    condition = output.complex == var.bar
    error_message = "expected to fail"
  }
}

run "validate_complex_output_sensitive" {
  // the rhs is sensitive
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
