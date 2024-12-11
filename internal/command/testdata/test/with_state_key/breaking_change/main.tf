resource "test_resource" "renamed_without_move" {
  value = "test"
}

output "test_id" {
  value = test_resource.renamed_without_move.id
}
