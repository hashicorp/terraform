package fastly

import (
	"fmt"
	"sort"
)

// DictionaryItem represents a dictionary item response from the Fastly API.
type DictionaryItem struct {
	ServiceID    string `mapstructure:"service_id"`
	DictionaryID string `mapstructure:"dictionary_id"`

	ItemKey   string `mapstructure:"item_key"`
	ItemValue string `mapstructure:"item_value"`
}

// dictionaryItemsByKey is a sortable list of dictionary items.
type dictionaryItemsByKey []*DictionaryItem

// Len, Swap, and Less implement the sortable interface.
func (s dictionaryItemsByKey) Len() int      { return len(s) }
func (s dictionaryItemsByKey) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s dictionaryItemsByKey) Less(i, j int) bool {
	return s[i].ItemKey < s[j].ItemKey
}

// ListDictionaryItemsInput is used as input to the ListDictionaryItems function.
type ListDictionaryItemsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Dictionary is the ID of the dictionary to retrieve items for (required).
	Dictionary string
}

// ListDictionaryItems returns the list of dictionary items for the
// configuration version.
func (c *Client) ListDictionaryItems(i *ListDictionaryItemsInput) ([]*DictionaryItem, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Dictionary == "" {
		return nil, ErrMissingDictionary
	}

	path := fmt.Sprintf("/service/%s/dictionary/%s/items", i.Service, i.Dictionary)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var bs []*DictionaryItem
	if err := decodeJSON(&bs, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(dictionaryItemsByKey(bs))
	return bs, nil
}

// CreateDictionaryItemInput is used as input to the CreateDictionaryItem function.
type CreateDictionaryItemInput struct {
	// Service is the ID of the service. Dictionary is the ID of the dictionary.
	// Both fields are required.
	Service    string
	Dictionary string

	ItemKey   string `form:"item_key,omitempty"`
	ItemValue string `form:"item_value,omitempty"`
}

// CreateDictionaryItem creates a new Fastly dictionary item.
func (c *Client) CreateDictionaryItem(i *CreateDictionaryItemInput) (*DictionaryItem, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Dictionary == "" {
		return nil, ErrMissingDictionary
	}

	path := fmt.Sprintf("/service/%s/dictionary/%s/item", i.Service, i.Dictionary)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *DictionaryItem
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// GetDictionaryItemInput is used as input to the GetDictionaryItem function.
type GetDictionaryItemInput struct {
	// Service is the ID of the service. Dictionary is the ID of the dictionary.
	// Both fields are required.
	Service    string
	Dictionary string

	// ItemKey is the name of the dictionary item to fetch.
	ItemKey string
}

// GetDictionaryItem gets the dictionary item with the given parameters.
func (c *Client) GetDictionaryItem(i *GetDictionaryItemInput) (*DictionaryItem, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Dictionary == "" {
		return nil, ErrMissingDictionary
	}

	if i.ItemKey == "" {
		return nil, ErrMissingItemKey
	}

	path := fmt.Sprintf("/service/%s/dictionary/%s/item/%s", i.Service, i.Dictionary, i.ItemKey)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *DictionaryItem
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateDictionaryItemInput is used as input to the UpdateDictionaryItem function.
type UpdateDictionaryItemInput struct {
	// Service is the ID of the service. Dictionary is the ID of the dictionary.
	// Both fields are required.
	Service    string
	Dictionary string

	// ItemKey is the name of the dictionary item to fetch.
	ItemKey string

	ItemValue string `form:"item_value,omitempty"`
}

// UpdateDictionaryItem updates a specific dictionary item.
func (c *Client) UpdateDictionaryItem(i *UpdateDictionaryItemInput) (*DictionaryItem, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Dictionary == "" {
		return nil, ErrMissingDictionary
	}

	if i.ItemKey == "" {
		return nil, ErrMissingItemKey
	}

	path := fmt.Sprintf("/service/%s/dictionary/%s/item/%s", i.Service, i.Dictionary, i.ItemKey)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *DictionaryItem
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteDictionaryItemInput is the input parameter to DeleteDictionaryItem.
type DeleteDictionaryItemInput struct {
	// Service is the ID of the service. Dictionary is the ID of the dictionary.
	// Both fields are required.
	Service    string
	Dictionary string

	// ItemKey is the name of the dictionary item to delete.
	ItemKey string
}

// DeleteDictionaryItem deletes the given dictionary item.
func (c *Client) DeleteDictionaryItem(i *DeleteDictionaryItemInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Dictionary == "" {
		return ErrMissingDictionary
	}

	if i.ItemKey == "" {
		return ErrMissingItemKey
	}

	path := fmt.Sprintf("/service/%s/dictionary/%s/item/%s", i.Service, i.Dictionary, i.ItemKey)
	_, err := c.Delete(path, nil)
	if err != nil {
		return err
	}

	// Unlike other endpoints, the dictionary endpoint does not return a status
	// response - it just returns a 200 OK.
	return nil
}
