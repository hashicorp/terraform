
provider "foo" {
}

provider "bar" {
  other = 12
}

provider "bar" {
  other = 13

  alias = "bar"
}
