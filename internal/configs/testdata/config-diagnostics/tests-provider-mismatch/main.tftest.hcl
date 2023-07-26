
provider "foo" {}

provider "foo" {
  alias = "bar"
}

provider "bar" {
  alias = "foo"
}

run "default_should_be_fine" {}

run "bit_complicated_still_okay "{

  providers = {
    foo = foo
    foo.bar = foo.bar
    bar = bar.foo
  }

}

run "mismatched_foo_direct" {

  providers = {
    foo = bar // bad!
    foo.bar = foo.bar
    bar = bar.foo
  }

}
