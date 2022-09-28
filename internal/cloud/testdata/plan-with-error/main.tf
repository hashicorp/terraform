resource "null_resource" "foo" {
  triggers {
    random = "${guid()}"
  }
}
