module "child" {
    source = "./mod"
}

import {
    to = module.child.test_object.foo
    id = "rootimport"
}