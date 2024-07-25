provider "test" {
  resource_prefix = run.main.resource_directory
}

provider "test" {
  alias = "usable"
}

run "main" {
  providers = {
    test = test.usable
  }

  variables {
    resource_directory = "resource"
  }
}
