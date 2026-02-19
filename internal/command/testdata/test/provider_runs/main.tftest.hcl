variables {
  resource_directory = "resources"
}

provider "test" {
  alias = "setup"
  resource_prefix = var.resource_directory
}

run "setup" {
  module {
    source = "./setup"
  }

  providers = {
    test = test.setup
  }
}

provider "test" {
  resource_prefix = run.setup.resource_directory
}

run "main" {}
