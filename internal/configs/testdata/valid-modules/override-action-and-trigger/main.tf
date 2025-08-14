action "test_action" "test" {
    foo = "bar"
}

action "test_action" "unchanged" {
    foo = "bar"
}

resource "test_instance" "test" {
    lifecycle {
      action_trigger {
        events = [after_create, after_update]
        actions = [action.test_action.dosomething]
      }
    }
}