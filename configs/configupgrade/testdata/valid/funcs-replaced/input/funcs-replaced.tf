locals {
  list        = "${list("a", "b", "c")}"
  list_concat = "${concat(list("a", "b"), list("c"))}"
  list_empty  = "${list()}"

  map         = "${map("a", "b", "c", "d")}"
  map_merge   = "${merge(map("a", "b"), map("c", "d"))}"
  map_empty   = "${map()}"
  map_invalid = "${map("a", "b", "c")}"

  list_of_map = "${list(map("a", "b"))}"
  map_of_list = "${map("a", list("b"))}"

  lookup_literal = "${lookup(map("a", "b"), "a")}"
  lookup_ref     = "${lookup(local.map, "a")}"

  # Undocumented HIL implementation details that some users nonetheless relied on.
  conv_bool_to_string  = "${__builtin_BoolToString(true)}"
  conv_float_to_int    = "${__builtin_FloatToInt(1.5)}"
  conv_float_to_string = "${__builtin_FloatToString(1.5)}"
  conv_int_to_float    = "${__builtin_IntToFloat(1)}"
  conv_int_to_string   = "${__builtin_IntToString(1)}"
  conv_string_to_int   = "${__builtin_StringToInt("1")}"
  conv_string_to_float = "${__builtin_StringToFloat("1.5")}"
  conv_string_to_bool  = "${__builtin_StringToBool("true")}"
}
