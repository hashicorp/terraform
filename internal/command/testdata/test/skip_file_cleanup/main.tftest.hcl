test {
  skip_cleanup = true
}

run "test" {
  variables {
    id = "test"
  }
}

run "test_two" {
  variables {
    id = "test_two"
  }
}

run "test_three" {
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
  skip_cleanup = false # This will be cleaned up, and test_four will not
  variables {
    id = "test_five"
  }
}