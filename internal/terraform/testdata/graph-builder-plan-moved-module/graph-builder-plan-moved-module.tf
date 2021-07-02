terraform {
  experiments = [config_driven_move]
}

module "child" {
  source = "./child"
  count  = 1
}

resource "test_object" "a" {
}

resource "test_object" "b" {
}

moved {
  from = test_object.a
  to   = module.child[0].test_object.a
}

moved {
  from = module.child[0].test_object.b
  to   = test_object.b
}

moved {
  from = module.child
  to   = module.blessed_child
}
