resource "test_instance" "first_many" {
  count = 2
}

resource "test_instance" "one" {
  image = "${test_instance.first_many.*.id[0]}"
}

resource "test_instance" "splat_of_one" {
  image = "${test_instance.one.*.id[0]}"
}

resource "test_instance" "second_many" {
  count = "${length(test_instance.first_many)}"
  security_groups = "${test_instance.first_many.*.id[count.index]}"
}
