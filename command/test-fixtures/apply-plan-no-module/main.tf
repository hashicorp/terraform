resource "test_instance" "tmpl" {
  foo = "${file("${path.module}/template.txt")}"
}

output "template" {
	value = "${test_instance.tmpl.foo}"
}
