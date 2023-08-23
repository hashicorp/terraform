terraform {
	cloud {
		organization = "hashicorp"

		workspaces {
			name = "test"
		}
	}
}
