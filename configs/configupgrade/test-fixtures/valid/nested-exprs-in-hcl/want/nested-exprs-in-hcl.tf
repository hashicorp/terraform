locals {
  in_map = {
    foo = var.baz
  }
  in_list = [
    var.baz,
    var.bar,
  ]
  in_list_oneline = [var.baz, var.bar]
  in_map_of_list = {
    foo = [var.baz]
  }
  in_list_of_map = [
    {
      foo = var.baz
    },
  ]
}
