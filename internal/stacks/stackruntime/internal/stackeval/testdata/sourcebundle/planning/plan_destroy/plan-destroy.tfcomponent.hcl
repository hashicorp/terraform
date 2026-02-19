
required_providers {
  test = {
    source = "terraform.io/builtin/test"
  }
}

component "a" {
  source = "./module_a"

  providers = {
    test = provider.test.main
  }
}

component "b" {
  source = "./module_b"

  inputs = {
    from_a = component.a.result
  }

  providers = {
    test = provider.test.main
  }
}

provider "test" "main" {
}

output "from_a" {
  type  = string
  value = component.a.result
}

output "from_b" {
  type  = string
  value = component.b.result
}
