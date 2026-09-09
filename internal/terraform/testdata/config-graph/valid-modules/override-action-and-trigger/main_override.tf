action "test_action" "test" {
    config {
      foo = "baz"
    }
}

resource "test_instance" "test" {
    lifecycle {
      action_trigger {
        events = [before_create]
        actions = [action.test_action.dosomething]
      }
    }
}