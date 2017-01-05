package clever

// common to applications and addons
type Env struct {
	Key   string `json:"name"`
	Value string `json:"value"`
}

type Fqdn struct {
	Fqdn string `json:"fqdn"`
}
