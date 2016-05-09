package cloudflare

// Railgun

// https://api.cloudflare.com/#railgun-create-railgun
// POST /railguns
func (c *API) CreateRailgun() {
}

// https://api.cloudflare.com/#railgun-railgun-details
// GET /railguns/:identifier

// https://api.cloudflare.com/#railgun-get-zones-connected-to-a-railgun
// GET /railguns/:identifier/zones

// https://api.cloudflare.com/#railgun-enable-or-disable-a-railgun
// PATCH /railguns/:identifier

// https://api.cloudflare.com/#railgun-delete-railgun
// DELETE /railguns/:identifier

// Zone railgun info

// https://api.cloudflare.com/#railguns-for-a-zone-get-available-railguns
// GET /zones/:zone_identifier/railguns
func (c *API) Railguns() {
}

// https://api.cloudflare.com/#railguns-for-a-zone-get-railgun-details
// GET /zones/:zone_identifier/railguns/:identifier
func (c *API) Railgun() {
}

// https://api.cloudflare.com/#railguns-for-a-zone-connect-or-disconnect-a-railgun
// PATCH /zones/:zone_identifier/railguns/:identifier
func (c *API) ZoneRailgun(connected bool) {
}
