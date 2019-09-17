// Test note: the configupgrade tool will ignore this possibly-relative module
// source because it does not find a local directory "foo". The example where
// the configupgrade tool makes a recommendation about relative module sources
// is is in relative-module-source.
module "foo" {
  source = "foo"
}
