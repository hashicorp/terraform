run "old_version" {
  plan_options {
    state_alias = "test1"
  }
}

run "new_code" {
  module {
    source = "./breaking_change"
  }
  plan_options {
    state_alias = "test1"
  }
  assert {
    condition = test_resource.renamed_without_move.id == run.old_version.test_id
    error_message = "resource renamed without moved block"
  }
}
