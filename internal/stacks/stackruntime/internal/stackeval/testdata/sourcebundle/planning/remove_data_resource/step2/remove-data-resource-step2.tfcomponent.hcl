required_providers {
  test = {
    source = "terraform.io/builtin/test"
  }
}

provider "test" "main" {
}

component "main" {
  source = "./"

  providers = {
    test = provider.test.main
  }
}
