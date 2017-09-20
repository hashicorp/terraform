provider "foo" {
  version = ">=1.0.0"
}

provider "foo" {
  version = ">=2.0.0"
  alias = "bar"
}
