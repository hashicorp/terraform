
provider "foo" {
}

provider "bar" {
  version = ">= 1.0.2"

  other = 12
}

provider "bar" {
  other = 13

  alias = "bar"
}
