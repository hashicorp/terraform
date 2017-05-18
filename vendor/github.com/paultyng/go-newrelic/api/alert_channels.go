package api

import (
	"fmt"
	"net/url"
)

func (c *Client) queryAlertChannels() ([]AlertChannel, error) {
	channels := []AlertChannel{}

	reqURL, err := url.Parse("/alerts_channels.json")
	if err != nil {
		return nil, err
	}

	nextPath := reqURL.String()

	for nextPath != "" {
		resp := struct {
			Channels []AlertChannel `json:"channels,omitempty"`
		}{}

		nextPath, err = c.Do("GET", nextPath, nil, &resp)
		if err != nil {
			return nil, err
		}

		channels = append(channels, resp.Channels...)
	}

	return channels, nil
}

// GetAlertChannel returns a specific alert channel by ID
func (c *Client) GetAlertChannel(id int) (*AlertChannel, error) {
	channels, err := c.queryAlertChannels()
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		if channel.ID == id {
			return &channel, nil
		}
	}

	return nil, ErrNotFound
}

// ListAlertChannels returns all alert policies for the account.
func (c *Client) ListAlertChannels() ([]AlertChannel, error) {
	return c.queryAlertChannels()
}

// CreateAlertChannel allows you to create an alert channel with the specified data and links.
func (c *Client) CreateAlertChannel(channel AlertChannel) (*AlertChannel, error) {
	// TODO: support attaching policy ID's here?
	// qs := map[string]string{
	// 	"policy_ids[]": channel.Links.PolicyIDs,
	// }

	if len(channel.Links.PolicyIDs) > 0 {
		return nil, fmt.Errorf("cannot create an alert channel with policy IDs, you must attach polidy IDs after creation")
	}

	req := struct {
		Channel AlertChannel `json:"channel"`
	}{
		Channel: channel,
	}

	resp := struct {
		Channels []AlertChannel `json:"channels,omitempty"`
	}{}

	_, err := c.Do("POST", "/alerts_channels.json", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Channels[0], nil
}

// DeleteAlertChannel deletes the alert channel with the specified ID.
func (c *Client) DeleteAlertChannel(id int) error {
	u := &url.URL{Path: fmt.Sprintf("/alerts_channels/%v.json", id)}
	_, err := c.Do("DELETE", u.String(), nil, nil)
	return err
}
