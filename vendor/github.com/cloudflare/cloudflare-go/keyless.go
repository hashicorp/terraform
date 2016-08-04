package cloudflare

// CreateKeyless creates a new Keyless SSL configuration for the zone.
// API reference:
// 	https://api.cloudflare.com/#keyless-ssl-for-a-zone-create-a-keyless-ssl-configuration
// 	POST /zones/:zone_identifier/keyless_certificates
func (api *API) CreateKeyless() {
}

// ListKeyless lists Keyless SSL configurations for a zone.
// API reference:
// 	https://api.cloudflare.com/#keyless-ssl-for-a-zone-list-keyless-ssls
// 	GET /zones/:zone_identifier/keyless_certificates
func (api *API) ListKeyless() {
}

// Keyless provides the configuration for a given Keyless SSL identifier.
// API reference:
// 	https://api.cloudflare.com/#keyless-ssl-for-a-zone-keyless-ssl-details
// 	GET /zones/:zone_identifier/keyless_certificates/:identifier
func (api *API) Keyless() {
}

// UpdateKeyless updates an existing Keyless SSL configuration.
// API reference:
// 	https://api.cloudflare.com/#keyless-ssl-for-a-zone-update-keyless-configuration
// 	PATCH /zones/:zone_identifier/keyless_certificates/:identifier
func (api *API) UpdateKeyless() {
}

// DeleteKeyless deletes an existing Keyless SSL configuration.
// API reference:
// 	https://api.cloudflare.com/#keyless-ssl-for-a-zone-delete-keyless-configuration
// 	DELETE /zones/:zone_identifier/keyless_certificates/:identifier
func (api *API) DeleteKeyless() {
}
