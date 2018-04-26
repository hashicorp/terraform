resource "aws_instance" "foo" {
  id = "foo"
  val = "${module.modb.foo}"
}

module "modb" {
  source = "./modb"
}

output "foo" {
  value = "${aws_instance.foo.id}"
}

output "from_modb" {
  value = "${module.modb.foo}"
}
