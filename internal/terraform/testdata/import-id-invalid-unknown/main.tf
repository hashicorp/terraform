resource "test_resource" "foo" {

}

import {
  to = test_resource.bar
  id = test_resource.foo.id
}

resource "test_resource" "bar" {

}
