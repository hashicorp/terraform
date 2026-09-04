resource "test_instance" "foo" {
  ami = "bar"
}

resource "test_instance" "excludeme" {
  ami = "excluded"
}

resource "test_instance" "dependent_on_excludeme" {
  ami = test_instance.excludeme.id
}
