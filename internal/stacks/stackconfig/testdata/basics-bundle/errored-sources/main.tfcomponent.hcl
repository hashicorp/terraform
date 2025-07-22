required_providers {
  null = {
    source  = "hashicorp/null"
    version = "3.2.1"
  }
}

provider "null" "a" {}

removed {
  from = stack.a.component.a // bad, stack.a is undefined so this is orphaned

  source = "./"

  providers = {
    null = provider.null.a
  }
}

removed {
  from = stack.a.stack.b // bad, stack.a is undefined so this is orphaned
  source = "./subdir"
}

removed {
  from = stack.b["a"]
  source = "./subdir"
}

removed {
  from = stack.b["b"]
  source = "./" // bad, the sources should be the same for stack.b
}

removed {
  from = stack.a.component.b["a"]

  source = "./"

  providers = {
    null = provider.null.a
  }
}

removed {
  from = stack.a.component.b["b"] // bad, the sources should be the same for component.b

  source = "./subdir"

  providers = {
    null = provider.null.a
  }
}