resource "null_resource" "write" {
	foo = "attribute"
}

data "null_data_source" "read" {
	foo = ""
	depends_on = ["null_resource.write"]
}
