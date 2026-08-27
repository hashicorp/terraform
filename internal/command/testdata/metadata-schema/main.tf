terraform {
    required_providers {
        test = {source = "hashicorp/test" }
        bufo = { source = "austinvalle/bufo" } 
        anotherone = { source = "hashicorp/example" }
    }
}

resource "test_object" "test" {}