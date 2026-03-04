# Quoted addresses (issue #34041)
moved {
  from = "module.foo"
  to   = "module.bar"
}

# Unqualified resource names without type prefix (issue #34162)
moved {
  from = bar
  to   = foo
}

import {
  to = test_instance.foo
  id = "test"
}
