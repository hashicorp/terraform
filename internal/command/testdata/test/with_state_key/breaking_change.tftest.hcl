run "old_version" {
  state_key = "test1"
}

run "new_code" {
  state_key = "test1"
  module {
    source = "./breaking_change"
  }
  assert {
    condition = test_resource.renamed_without_move.id == run.old_version.test_id
    error_message = "resource renamed without moved block"
  }
}
