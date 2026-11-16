action "test_action" "test" {}

ephemeral "test_ephemeral" "test" {
  lifecycle {
    action_trigger {
      actions = [action.test_action.test]
      events  = [after_create, after_update]
    }
  }
}
