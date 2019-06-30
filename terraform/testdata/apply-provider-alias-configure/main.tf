provider "another" {
  foo = "bar"
}

provider "another" {
  alias = "two"
  foo   = "bar"
}

resource "another_instance" "foo" {}

resource "another_instance" "bar" {
  provider = "another.two"
}
