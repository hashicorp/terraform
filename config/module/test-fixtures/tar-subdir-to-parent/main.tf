module "foo" {
	// the module in sub references sibling module baz via "../baz"
    source = "./foo.tgz//sub"
}
