variable "num" {
}

resource "test_thing" "source" {
  count = "${var.num}"

  # The diffFunc in the test exports "name" here too, which we can use
  # to test values that are known during plan.
}

resource "test_thing" "multi_count_var" {
  count = "${var.num}"

  # Can pluck a single item out of a multi-var
  source_id   = "${test_thing.source.*.id[count.index]}"
  source_name = "${test_thing.source.*.name[count.index]}"
}

resource "test_thing" "multi_count_derived" {
  # Can use the source to get the count
  count = "${length(test_thing.source)}"

  source_id   = "${test_thing.source.*.id[count.index]}"
  source_name = "${test_thing.source.*.name[count.index]}"
}

resource "test_thing" "whole_splat" {
  # Can "splat" the ids directly into an attribute of type list.
  source_ids   = "${test_thing.source.*.id}"
  source_names = "${test_thing.source.*.name}"

  # Accessing through a function should work.
  source_ids_from_func   = "${split(" ", join(" ", test_thing.source.*.id))}"
  source_names_from_func = "${split(" ", join(" ", test_thing.source.*.name))}"

  # A common pattern of selecting with a default.
  first_source_id   = "${element(concat(test_thing.source.*.id, list("default")), 0)}"
  first_source_name = "${element(concat(test_thing.source.*.name, list("default")), 0)}"

  # Legacy form: Prior to Terraform having comprehensive list support,
  # splats were treated as a special case and required to be presented
  # in a wrapping list. This is no longer the suggested form, but we
  # need it to keep working for compatibility.
  #
  # This should result in exactly the same result as the above, even
  # though it looks like it would result in a list of lists.
  source_ids_wrapped   = ["${test_thing.source.*.id}"]
  source_names_wrapped = ["${test_thing.source.*.name}"]

}

module "child" {
  source = "./child"

  num          = "${var.num}"
  source_ids   = "${test_thing.source.*.id}"
  source_names = "${test_thing.source.*.name}"
}

output "source_ids" {
  value = "${test_thing.source.*.id}"
}

output "source_names" {
  value = "${test_thing.source.*.name}"
}
