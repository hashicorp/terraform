variable "input" {
	type = string
	default = "foo"
}

list "test_instance" "test" {
	provider = test

	config {
		ami = var.input
	}
}

list "test_instance" "test2" {
	provider = test
	
	config {
  	// this traversal is invalid for a list resource
  	ami = list.test_instance.test.state.instance_type
	}
}
