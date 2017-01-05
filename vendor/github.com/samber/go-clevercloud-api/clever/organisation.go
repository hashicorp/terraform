package clever

type Organisation struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (c *Client) GetOrganisation() (*Organisation, error) {
	org := &Organisation{}

	err := c.get("/organisations/"+c.config.OrgId, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}
