package api

import (
	"net/url"
	"regexp"
	"strconv"
)

// UpdateAlertPolicyChannels updates a policy by adding the specified notification channels.
func (c *Client) UpdateAlertPolicyChannels(policyID int, channelIDs []int) error {
	channelIDStrings := make([]string, len(channelIDs))

	for i, channelID := range channelIDs {
		channelIDStrings[i] = strconv.Itoa(channelID)
	}

	reqURL, err := url.Parse("/alerts_policy_channels.json")
	if err != nil {
		return err
	}

	qs := url.Values{
		"policy_id":   []string{strconv.Itoa(policyID)},
		"channel_ids": channelIDStrings,
	}
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	_, err = c.Do("PUT", nextPath, nil, nil)
	return err
}

// DeleteAlertPolicyChannel deletes a notification channel from an alert policy.
func (c *Client) DeleteAlertPolicyChannel(policyID int, channelID int) error {
	reqURL, err := url.Parse("/alerts_policy_channels.json")
	if err != nil {
		return err
	}

	qs := url.Values{
		"policy_id":  []string{strconv.Itoa(policyID)},
		"channel_id": []string{strconv.Itoa(channelID)},
	}
	reqURL.RawQuery = qs.Encode()

	nextPath := reqURL.String()

	_, err = c.Do("DELETE", nextPath, nil, nil)
	if err != nil {
		if apiErr, ok := err.(*ErrorResponse); ok {
			matched, err := regexp.MatchString("Alerts policy with ID: \\d+ is not valid.", apiErr.Detail.Title)
			if err != nil {
				return err
			}

			if matched {
				return ErrNotFound
			}
		}

		return err
	}

	return nil
}
