provider "test" {
  resource_prefix = run.missing.resource_directory
}

run "main" {
  variables {
    resource_directory = "resource"
  }
}
