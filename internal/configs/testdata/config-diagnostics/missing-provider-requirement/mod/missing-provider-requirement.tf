# Because of our backward-compatibility concession of allowing use of some
# long-standing official providers without explicitly declaring a dependency
# on them, module authors are sometimes surprised the first time they try
# to use a partner or community provider.
#
# This test covers the error message we generate in that case, which is intended
# to make it clear which resource declaration is causing the problem and that
# it can be solved by adding an explicit dependency in the required_providers
# block of the module.

resource "unofficial_thingy" "example" {
}
