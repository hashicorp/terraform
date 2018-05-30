variable "num" {
}

resource "test_thing" "source" {
  count = var.num

  # The diffFunc in the test exports "name" here too, which we can use
  # to test values that are known during plan.
}

resource "test_thing" "multi_count_var" {
  count = var.num

  # Can pluck a single item out of a multi-var
  source_id   = test_thing.source.*.id[count.index]
  source_name = test_thing.source.*.name[count.index]
}

resource "test_thing" "multi_count_derived" {
  # Can use the source to get the count
  count = length(test_thing.source)

  source_id   = test_thing.source.*.id[count.index]
  source_name = test_thing.source.*.name[count.index]
}

resource "test_thing" "whole_splat" {
  # Can "splat" the ids directly into an attribute of type list.
  source_ids   = test_thing.source.*.id
  source_names = test_thing.source.*.name

  # Accessing through a function should work.
  source_ids_from_func   = split(" ", join(" ", test_thing.source.*.id))
  source_names_from_func = split(" ", join(" ", test_thing.source.*.name))

  # A common pattern of selecting with a default.
  first_source_id   = element(concat(test_thing.source.*.id, ["default"]), 0)
  first_source_name = element(concat(test_thing.source.*.name, ["default"]), 0)

  # Prior to v0.12 we were handling lists containing list interpolations as
  # a special case, flattening the result, for compatibility with behavior
  # prior to v0.10. This deprecated handling is now removed, and so these
  # each produce a list of lists. We're still using the interpolation syntax
  # here, rather than the splat expression directly, to properly mimic how
  # this would've looked prior to v0.12 to be explicit about what the new
  # behavior is for this old syntax.
  source_ids_wrapped   = ["${test_thing.source.*.id}"]
  source_names_wrapped = ["${test_thing.source.*.name}"]

}

module "child" {
  source = "./child"

  num          = var.num
  source_ids   = test_thing.source.*.id
  source_names = test_thing.source.*.name
}

output "source_ids" {
  value = test_thing.source.*.id
}

output "source_names" {
  value = test_thing.source.*.name
}
