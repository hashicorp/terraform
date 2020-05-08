resource "null_instance" "write" {
	foo = "attribute"
}

data "null_data_source" "read" {
	depends_on = ["null_instance.write"]
}
