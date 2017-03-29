resource "null_resource" "var" {
    key = "${module.unknown.value}"
}
