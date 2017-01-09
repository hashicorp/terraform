package arukas

func displayVersion() {
	client = NewClientWithOsExitOnErr()
	client.Println(nil, VERSION)
}
