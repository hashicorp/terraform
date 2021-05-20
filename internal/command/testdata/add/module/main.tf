terraform {
    required_providers  {
        // This is deliberately odd, so we can test that the correct happycorp
        // provider is selected for any test_ resource added for this module
        test = {
            source = "happycorp/test"
        }
    }
}

resource "test_instance" "exists" {
    // I exist!
}

module "child" {
    source = "./module"
}