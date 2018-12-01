locals {
  list        = ["a", "b", "c"]
  list_concat = concat(["a", "b"], ["c"])
  list_empty  = []

  map = {
    "a" = "b"
    "c" = "d"
  }
  map_merge = merge(
    {
      "a" = "b"
    },
    {
      "c" = "d"
    },
  )
  map_empty   = {}
  map_invalid = map("a", "b", "c")

  list_of_map = [
    {
      "a" = "b"
    },
  ]
  map_of_list = {
    "a" = ["b"]
  }

  lookup_literal = {
    "a" = "b"
  }["a"]
  lookup_ref = local.map["a"]
}
