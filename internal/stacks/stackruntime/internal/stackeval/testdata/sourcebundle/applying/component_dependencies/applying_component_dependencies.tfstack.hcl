required_providers {
  test = {
    source = "terraform.io/builtin/test"
  }
}

provider "test" "main" {
}

component "a" {
  source = "./"

  inputs = {
    marker = "a"
  }
  providers = {
    test = provider.test.main
  }
}

component "b" {
  source   = "./"
  for_each = toset(["i", "ii", "iii"])

  inputs = {
    marker = "b.${each.key}"
    deps   = [component.a.marker]
  }
  providers = {
    test = provider.test.main
  }
}

component "c" {
  source = "./"

  inputs = {
    marker = "c"
    deps   = [ for b in component.b : b.marker ]
  }
  providers = {
    test = provider.test.main
  }
}
