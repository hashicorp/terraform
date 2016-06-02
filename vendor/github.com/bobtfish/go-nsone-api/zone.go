package nsone

type ZoneSecondaryServer struct {
	Ip     string `json:"ip"`
	Port   int    `json:"port,omitempty"`
	Notify bool   `json:"notify"`
}

type ZonePrimary struct {
	Enabled     bool                  `json:"enabled"`
	Secondaries []ZoneSecondaryServer `json:"secondaries"`
}

type ZoneSecondary struct {
	Status       string `json:"status,omitempty"`
	Last_xfr     int    `json:"last_xfr,omitempty"`
	Primary_ip   string `json:"primary_ip,omitempty"`
	Primary_port int    `json:"primary_port,omitempty"`
	Enabled      bool   `json:"enabled"`
	Expired      bool   `json:"expired,omitempty"`
}

type Zone struct {
	Id            string            `json:"id,omitempty"`
	Ttl           int               `json:"ttl,omitempty"`
	Nx_ttl        int               `json:"nx_ttl,omitempty"`
	Retry         int               `json:"retry,omitempty"`
	Zone          string            `json:"zone,omitempty"`
	Refresh       int               `json:"refresh,omitempty"`
	Expiry        int               `json:"expiry,omitempty"`
	Primary       *ZonePrimary      `json:"primary,omitempty"`
	Dns_servers   []string          `json:"dns_servers,omitempty"`
	Networks      []int             `json:"networks,omitempty"`
	Network_pools []string          `json:"network_pools,omitempty"`
	Hostmaster    string            `json:"hostmaster,omitempty"`
	Pool          string            `json:"pool,omitempty"`
	Meta          map[string]string `json:"meta,omitempty"`
	Secondary     *ZoneSecondary    `json:"secondary,omitempty"`
	Link          string            `json:"link,omitempty"`
}

func NewZone(zone string) *Zone {
	z := Zone{
		Zone: zone,
	}
	z.MakePrimary()
	return &z
}

func (z *Zone) MakePrimary(secondaries ...ZoneSecondaryServer) {
	z.Secondary = nil
	z.Primary = &ZonePrimary{
		Enabled:     true,
		Secondaries: secondaries,
	}
	if z.Primary.Secondaries == nil {
		z.Primary.Secondaries = make([]ZoneSecondaryServer, 0)
	}
}

func (z *Zone) MakeSecondary(ip string) {
	z.Secondary = &ZoneSecondary{
		Enabled:      true,
		Primary_ip:   ip,
		Primary_port: 53,
	}
	s := make([]ZoneSecondaryServer, 0)
	z.Primary = &ZonePrimary{
		Enabled:     false,
		Secondaries: s,
	}
}

func (z *Zone) LinkTo(to string) {
	z.Meta = nil
	z.Ttl = 0
	z.Nx_ttl = 0
	z.Retry = 0
	z.Refresh = 0
	z.Expiry = 0
	z.Primary = nil
	z.Dns_servers = nil
	z.Networks = nil
	z.Network_pools = nil
	z.Hostmaster = ""
	z.Pool = ""
	z.Secondary = nil
	z.Link = to
}
