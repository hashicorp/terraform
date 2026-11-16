run "test" {
  variables {
    id = "test"
  }
}

run "test_two" {
  skip_cleanup = true
  variables {
    id = "test_two"
  }
}

run "test_three" {
  skip_cleanup = true
  variables {
    id = "test_three"
  }
}

run "test_four" {
  variables {
    id = "test_four"
  }
}

run "test_five" {
  variables {
    id = "test_five"
  }
}