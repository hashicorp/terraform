resource "test_instance" "a" {
  subnet_ids = ["boop"] # this attribute takes a set of strings
}

output "b" {
  value = "${element(test_instance.a.subnet_ids, 0)}"
}
