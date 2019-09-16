
terraform {
  backend "example" {
    foo = "bar"

    baz {
      bar = "foo"
    }
  }
}
