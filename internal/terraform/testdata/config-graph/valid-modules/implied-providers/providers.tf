terraform {
  required_providers {
    // This is an expected "real world" example of a community provider, which
    // has resources named "foo_*" and will likely be used in configurations
    // with the local name of "foo".
    foo = {
      source = "registry.acme.corp/acme/foo"
    }

    // However, implied provider lookups are based on local name, not provider
    // type, and this example clarifies that. Only resources with addresses
    // starting "whatever_" will be assigned this provider implicitly.
    //
    // This is _not_ a recommended usage pattern. The best practice is for
    // local name and type to be the same, and only use a different local name
    // if there are provider type collisions.
    whatever = {
      source = "acme/something"
    }
  }
}
