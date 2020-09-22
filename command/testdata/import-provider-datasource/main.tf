provider "test" {
  foo = data.test_data.key.id
}

provider "test" {
  alias = "credentials"
}

data "test_data" "key" {
  provider = test.credentials
}

resource "test_instance" "foo" {}
