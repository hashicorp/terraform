locals {
  # Arithmetic
  add             = 1 + 2
  sub             = 1 - 2
  mul             = 1 * 2
  mod             = 4 % 2
  and             = true && true
  or              = true || true
  equal           = 1 == 2
  not_equal       = 1 != 2
  less_than       = 1 < 2
  greater_than    = 1 > 2
  less_than_eq    = 1 <= 2
  greater_than_eq = 1 >= 2
  neg             = -local.add

  # Call
  call_no_args  = foo()
  call_one_arg  = foo(1)
  call_two_args = foo(1, 2)

  # Conditional
  cond = true ? 1 : 2

  # Index
  index_str = foo["a"]
  index_num = foo[1]

  # Variable Access
  var_access_single = foo
  var_access_dot    = foo.bar
  var_access_splat  = foo.bar.*.baz
}
