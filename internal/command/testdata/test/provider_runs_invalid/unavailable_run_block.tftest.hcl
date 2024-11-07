provider "test" {
  resource_prefix = run.main.resource_directory
}

run "main" {
  variables {
    resource_directory = "resource"
  }
}
