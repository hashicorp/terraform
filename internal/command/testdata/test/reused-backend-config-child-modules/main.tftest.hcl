# The "state/terraform.tfstate" local backend is used with the implicit internal state "./child-module"
run "test_1" {
  module {
    source = "./child-module"
  }

  variables {
    input = "foobar"
  }

  backend "local" {
    path = "state/terraform.tfstate"
  }
}

# The "state/terraform.tfstate" local backend is used with the implicit internal state "" (empty string == root module under test)
run "test_2" {

  backend "local" {
    path = "state/terraform.tfstate"
  }
}
