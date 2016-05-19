package cloudflare

// https://api.cloudflare.com/#custom-ssl-for-a-zone-create-ssl-configuration
// POST /zones/:zone_identifier/custom_certificates
func (c *API) CreateSSL() {
}

// https://api.cloudflare.com/#custom-ssl-for-a-zone-list-ssl-configurations
// GET /zones/:zone_identifier/custom_certificates
func (c *API) ListSSL() {
}

// https://api.cloudflare.com/#custom-ssl-for-a-zone-ssl-configuration-details
// GET /zones/:zone_identifier/custom_certificates/:identifier
func (c *API) SSLDetails() {
}

// https://api.cloudflare.com/#custom-ssl-for-a-zone-update-ssl-configuration
// PATCH /zones/:zone_identifier/custom_certificates/:identifier
func (c *API) UpdateSSL() {
}

// https://api.cloudflare.com/#custom-ssl-for-a-zone-re-prioritize-ssl-certificates
// PUT /zones/:zone_identifier/custom_certificates/prioritize
func (c *API) ReprioSSL() {
}

// https://api.cloudflare.com/#custom-ssl-for-a-zone-delete-an-ssl-certificate
// DELETE /zones/:zone_identifier/custom_certificates/:identifier
func (c *API) DeleteSSL() {
}
