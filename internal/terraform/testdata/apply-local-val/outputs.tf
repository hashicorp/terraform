# These are in a separate file to make sure config merging is working properly

output "result_1" {
  value = "${local.result_1}"
}

output "result_3" {
  value = "${local.result_3}"
}
