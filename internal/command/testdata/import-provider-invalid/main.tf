terraform {
	backend "local" {
		path = "imported.tfstate"
	}
}

provider "test" {
    foo = "bar"
}

resource "test_instance" "foo" {
}

resource "unknown_instance" "baz" {
}
