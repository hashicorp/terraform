module "child" {
  source = "./child"
}

resource "test_instance" "a" {}

import {
  to = test_instance.a
  id = "test-a"
}
