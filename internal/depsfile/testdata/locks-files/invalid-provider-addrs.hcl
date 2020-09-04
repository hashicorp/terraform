provider "" { # ERROR: Invalid provider source address

}

provider "hashicorp/aws" { # ERROR: Non-normalized provider source address

}

provider "aws" { # ERROR: Non-normalized provider source address

}

provider "too/many/parts/here" { # ERROR: Invalid provider source address

}

provider "Registry.terraform.io/example/example" { # ERROR: Non-normalized provider source address

}

provider "registry.terraform.io/eXample/example" { # ERROR: Non-normalized provider source address

}

provider "registry.terraform.io/example/Example" { # ERROR: Non-normalized provider source address

}

provider "this/one/okay" {
  version = "1.0.0"
}

provider "this/one/okay" { # ERROR: Duplicate provider lock
}
