run "old_version" {
  module {
    source = "./old_version"
  }
  plan_options {
    state_alias = "test1"
  }
}

run "new_code" {
  plan_options {
    state_alias = "test1"
  }
  assert {
    condition = test_resource.test_id_moved.id == run.old_version.test_id
    error_message = "ressource_id differed"
  }
}
