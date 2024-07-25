# removed resource - basic
removed {
  from = test_resource.foo
  lifecycle {
    destroy = false
  }
}

# removed resource - with module
removed {
  from = module.gone.test_resource.bar
  lifecycle {
    destroy = false
  }
}

# removed module - basic
removed {
  from = module.gone.module.gonechild
  lifecycle {
    destroy = false
  }
}

module "child" {
  source = "./child"
}

# removed resource - overridden from module 
removed {
  from = module.child.test_resource.baz
  lifecycle {
    destroy = true 
  }
}
