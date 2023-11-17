provider "aws" {}

mock_provider "aws" {}

provider "aws" {
  alias = "test"
}

mock_provider "aws" {
  alias = "test"
}

run "test" {}
