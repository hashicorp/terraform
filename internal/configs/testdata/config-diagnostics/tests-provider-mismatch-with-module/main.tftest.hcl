
provider "foo" {}

provider "foo" {
  alias = "bar"
}

provider "bar" {}

run "setup_module" {

  module {
    source = "./setup"
  }

}

run "main_module" {}
