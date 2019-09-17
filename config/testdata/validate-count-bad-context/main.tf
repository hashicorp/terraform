resource "aws_instance" "foo" {
}

output "no_count_in_output" {
  value = "${count.index}"
}

module "no_count_in_module" {
  source = "./child"
  somevar = "${count.index}"
}
