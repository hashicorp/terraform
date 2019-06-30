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

  # Undocumented HIL implementation details that some users nonetheless relied on.
  conv_bool_to_string  = tostring(tobool(true))
  conv_float_to_int    = floor(1.5)
  conv_float_to_string = tostring(tonumber(1.5))
  conv_int_to_float    = floor(1)
  conv_int_to_string   = tostring(floor(1))
  conv_string_to_int   = floor(tostring("1"))
  conv_string_to_float = tonumber(tostring("1.5"))
  conv_string_to_bool  = tobool(tostring("true"))
}
