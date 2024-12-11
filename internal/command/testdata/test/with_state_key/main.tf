resource "test_resource" "test_id_moved" {
}

output "test_id" {
  value = test_resource.test_id_moved.id
}

moved {
  from = test_resource.test_id
  to   = test_resource.test_id_moved
}
