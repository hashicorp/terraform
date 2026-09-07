run "backend" {
  command = apply

  backend "local" {
    path = "/tests/state/terraform.tfstate"
  }
}

run "skip_cleanup" {
  command = apply

  # Should warn us about the skip_cleanup option being set.
  skip_cleanup = true
}
