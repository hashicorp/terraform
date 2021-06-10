resource "foo_resource" "web" {
  for_each = toset("frontend", "backend")
}

moved {
  from = foo_resource.web["ui"]
  to = foo_resource.web["frontend"]
}

moved {
  from = foo_resource.web["api"]
  to = foo_resource.web["backend"]
}

data "foo_data" "alpha" {}

moved {
  from = data.foo_data.a
  to = data.foo_data.alpha
}
