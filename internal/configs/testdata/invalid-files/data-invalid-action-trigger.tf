action "test_action" "test" {}

data "test_data" "test" {
  lifecycle {
    action_trigger {
      actions = [action.test_action.test]
      events  = [after_create, after_update]
    }
  }
}
