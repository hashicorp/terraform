run "test" {
  command = apply

  backend "remote" {
    organization = "example_corp"
  }
}
