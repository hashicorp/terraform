package udnssdk

import (
	"fmt"
	"net/http"
)

// AccountsService provides access to account resources
type AccountsService struct {
	client *Client
}

// Account represents responses from the service
type Account struct {
	AccountName           string `json:"accountName"`
	AccountHolderUserName string `json:"accountHolderUserName"`
	OwnerUserName         string `json:"ownerUserName"`
	NumberOfUsers         int    `json:"numberOfUsers"`
	NumberOfGroups        int    `json:"numberOfGroups"`
	AccountType           string `json:"accountType"`
}

// AccountListDTO represents a account index response
type AccountListDTO struct {
	Accounts   []Account  `json:"accounts"`
	Resultinfo ResultInfo `json:"resultInfo"`
}

// AccountKey represents the string identifier of an Account
type AccountKey string

// URI generates the URI for an Account
func (k AccountKey) URI() string {
	uri := "accounts"
	if k != "" {
		uri = fmt.Sprintf("accounts/%s", k)
	}
	return uri
}

// AccountsURI generates the URI for Accounts collection
func AccountsURI() string {
	return "accounts"
}

// Select requests all Accounts of user
func (s *AccountsService) Select() ([]Account, *http.Response, error) {
	var ald AccountListDTO
	res, err := s.client.get(AccountsURI(), &ald)

	accts := []Account{}
	for _, t := range ald.Accounts {
		accts = append(accts, t)
	}
	return accts, res, err
}

// Find requests an Account by AccountKey
func (s *AccountsService) Find(k AccountKey) (Account, *http.Response, error) {
	var t Account
	res, err := s.client.get(k.URI(), &t)
	return t, res, err
}

// Delete requests deletion of an Account by AccountKey
func (s *AccountsService) Delete(k AccountKey) (*http.Response, error) {
	return s.client.delete(k.URI(), nil)
}
