# The "foobar" backend does not exist and isn't a removed backend either
run "test_invalid_backend" {
  variables {
    input = "foobar"
  }

  backend "foobar" {
  }
}
