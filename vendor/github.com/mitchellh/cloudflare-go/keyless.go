package cloudflare

// https://api.cloudflare.com/#keyless-ssl-for-a-zone-create-a-keyless-ssl-configuration
// POST /zones/:zone_identifier/keyless_certificates
func (c *API) CreateKeyless() {
}

// https://api.cloudflare.com/#keyless-ssl-for-a-zone-list-keyless-ssls
// GET /zones/:zone_identifier/keyless_certificates
func (c *API) ListKeyless() {
}

// https://api.cloudflare.com/#keyless-ssl-for-a-zone-keyless-ssl-details
// GET /zones/:zone_identifier/keyless_certificates/:identifier
func (c *API) Keyless() {
}

// https://api.cloudflare.com/#keyless-ssl-for-a-zone-update-keyless-configuration
// PATCH /zones/:zone_identifier/keyless_certificates/:identifier
func (c *API) UpdateKeyless() {
}

// https://api.cloudflare.com/#keyless-ssl-for-a-zone-delete-keyless-configuration
// DELETE /zones/:zone_identifier/keyless_certificates/:identifier
func (c *API) DeleteKeyless() {
}
