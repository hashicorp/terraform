resource "test_object" "foo" {}

import {
    to = test_object.foo
    id = "import"
}