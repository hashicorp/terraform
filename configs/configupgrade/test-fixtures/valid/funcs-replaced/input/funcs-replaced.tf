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
}
