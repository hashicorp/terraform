# In state with `ami = "foo"`, so this should be a regular update. The provider
# should not detect changes on refresh.
resource "test_instance" "no_refresh" {
  ami = "bar"
}

# In state with `ami = "refresh-me"`, but the provider will return
# `"refreshed"` after the refresh phase. The plan should show the drift
# (`"refresh-me"` to `"refreshed"`) and plan the update (`"refreshed"` to
# `"baz"`).
resource "test_instance" "should_refresh_with_move" {
  ami = "baz"
}

moved {
  from = test_instance.should_refresh
  to   = test_instance.should_refresh_with_move
}
