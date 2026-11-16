run "old_version" {
  state_key = "test1"
  module {
    source = "./old_version"
  }
}

run "new_code" {
  state_key = "test1"
  assert {
    condition = test_resource.test_id_moved.id == run.old_version.test_id
    error_message = "ressource_id differed"
  }
}
