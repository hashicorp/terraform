# (overridden in parent module)
removed {
  from = test_resource.baz
  lifecycle {
    destroy = false
  }
}

# removed resource - in module
removed {
  from = test_resource.boo
  lifecycle {
    destroy = true
  }
}

# removed module - in module
removed {
  from = module.grandchild
  lifecycle {
    destroy = false
  }
}
