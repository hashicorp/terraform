package pagerduty

// ListAbilityResponse is the response when calling the ListAbility API endpoint.
type ListAbilityResponse struct {
	Abilities []string `json:"abilities"`
}

// ListAbilities lists all abilities on your account.
func (c *Client) ListAbilities() (*ListAbilityResponse, error) {
	resp, err := c.get("/abilities")
	if err != nil {
		return nil, err
	}
	var result ListAbilityResponse
	return &result, c.decodeJSON(resp, &result)
}

// TestAbility Check if your account has the given ability.
func (c *Client) TestAbility(ability string) error {
	_, err := c.get("/abilities/" + ability)
	return err
}
