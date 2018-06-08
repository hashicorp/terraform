module "foo" {
	// the mock test registry will redirect this to the local tar file
    source = "registry/local/sub//baz"
}
