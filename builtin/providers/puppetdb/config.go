package puppetdb

// Config is the configuration parameters for a PuppetDB
type Config struct {
	URL  string
	Cert string
	Key  string
	CA   string
}
