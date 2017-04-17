package dnsimple

import (
	"fmt"
)

// RegistrarService handles communication with the registrar related
// methods of the DNSimple API.
//
// See https://developer.dnsimple.com/v2/registrar/
type RegistrarService struct {
	client *Client
}

// DomainCheck represents the result of a domain check.
type DomainCheck struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Premium   bool   `json:"premium"`
}

// DomainCheckResponse represents a response from a domain check request.
type DomainCheckResponse struct {
	Response
	Data *DomainCheck `json:"data"`
}

// CheckDomain checks a domain name.
//
// See https://developer.dnsimple.com/v2/registrar/#check
func (s *RegistrarService) CheckDomain(accountID, domainName string) (*DomainCheckResponse, error) {
	path := versioned(fmt.Sprintf("/%v/registrar/domains/%v/check", accountID, domainName))
	checkResponse := &DomainCheckResponse{}

	resp, err := s.client.get(path, checkResponse)
	if err != nil {
		return nil, err
	}

	checkResponse.HttpResponse = resp
	return checkResponse, nil
}

// DomainPremiumPrice represents the premium price for a premium domain.
type DomainPremiumPrice struct {
	// The domain premium price
	PremiumPrice string `json:"premium_price"`
	// The registrar action.
	// Possible values are registration|transfer|renewal
	Action string `json:"action"`
}

// DomainPremiumPriceResponse represents a response from a domain premium price request.
type DomainPremiumPriceResponse struct {
	Response
	Data *DomainPremiumPrice `json:"data"`
}

// DomainPremiumPriceOptions specifies the optional parameters you can provide
// to customize the RegistrarService.GetDomainPremiumPrice method.
type DomainPremiumPriceOptions struct {
	Action string `url:"action,omitempty"`
}

// Gets the premium price for a domain.
//
// You must specify an action to get the price for. Valid actions are:
// - registration
// - transfer
// - renewal
//
// See https://developer.dnsimple.com/v2/registrar/#premium-price
func (s *RegistrarService) GetDomainPremiumPrice(accountID, domainName string, options *DomainPremiumPriceOptions) (*DomainPremiumPriceResponse, error) {
	var err error
	path := versioned(fmt.Sprintf("/%v/registrar/domains/%v/premium_price", accountID, domainName))
	priceResponse := &DomainPremiumPriceResponse{}

	if options != nil {
		path, err = addURLQueryOptions(path, options)
		if err != nil {
			return nil, err
		}
	}

	resp, err := s.client.get(path, priceResponse)
	if err != nil {
		return nil, err
	}

	priceResponse.HttpResponse = resp
	return priceResponse, nil
}

// DomainRegisterRequest represents the attributes you can pass to a register API request.
// Some attributes are mandatory.
type DomainRegisterRequest struct {
	// The ID of the Contact to use as registrant for the domain
	RegistrantID int `json:"registrant_id"`
	// Set to true to enable the whois privacy service. An extra cost may apply.
	// Default to false.
	EnableWhoisPrivacy bool `json:"whois_privacy,omitempty"`
	// Set to true to enable the auto-renewal of the domain.
	// Default to true.
	EnableAutoRenewal bool `json:"auto_renew,omitempty"`
}

// DomainRegistration represents the result of a domain renewal call.
type DomainRegistration struct {
	ID           int    `json:"id"`
	DomainID     int    `json:"domain_id"`
	RegistrantID int    `json:"registrant_id"`
	Period       int    `json:"period"`
	State        string `json:"state"`
	AutoRenew    bool   `json:"auto_renew"`
	WhoisPrivacy bool   `json:"whois_privacy"`
	PremiumPrice string `json:"premium_price"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// DomainRegistrationResponse represents a response from an API method that results in a domain registration.
type DomainRegistrationResponse struct {
	Response
	Data *DomainRegistration `json:"data"`
}

// RegisterDomain registers a domain name.
//
// See https://developer.dnsimple.com/v2/registrar/#register
func (s *RegistrarService) RegisterDomain(accountID string, domainName string, request *DomainRegisterRequest) (*DomainRegistrationResponse, error) {
	path := versioned(fmt.Sprintf("/%v/registrar/domains/%v/registration", accountID, domainName))
	registrationResponse := &DomainRegistrationResponse{}

	// TODO: validate mandatory attributes RegistrantID

	resp, err := s.client.post(path, request, registrationResponse)
	if err != nil {
		return nil, err
	}

	registrationResponse.HttpResponse = resp
	return registrationResponse, nil
}

// DomainTransferRequest represents the attributes you can pass to a transfer API request.
// Some attributes are mandatory.
type DomainTransferRequest struct {
	// The ID of the Contact to use as registrant for the domain
	RegistrantID int `json:"registrant_id"`
	// The Auth-Code required to transfer the domain.
	// This is provided by the current registrar of the domain.
	AuthCode string `json:"auth_code,omitempty"`
	// Set to true to enable the whois privacy service. An extra cost may apply.
	// Default to false.
	EnableWhoisPrivacy bool `json:"whois_privacy,omitempty"`
	// Set to true to enable the auto-renewal of the domain.
	// Default to true.
	EnableAutoRenewal bool `json:"auto_renew,omitempty"`
}

// DomainTransfer represents the result of a domain renewal call.
type DomainTransfer struct {
	ID           int    `json:"id"`
	DomainID     int    `json:"domain_id"`
	RegistrantID int    `json:"registrant_id"`
	State        string `json:"state"`
	AutoRenew    bool   `json:"auto_renew"`
	WhoisPrivacy bool   `json:"whois_privacy"`
	PremiumPrice string `json:"premium_price"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// DomainTransferResponse represents a response from an API method that results in a domain transfer.
type DomainTransferResponse struct {
	Response
	Data *DomainTransfer `json:"data"`
}

// TransferDomain transfers a domain name.
//
// See https://developer.dnsimple.com/v2/registrar/#transfer
func (s *RegistrarService) TransferDomain(accountID string, domainName string, request *DomainTransferRequest) (*DomainTransferResponse, error) {
	path := versioned(fmt.Sprintf("/%v/registrar/domains/%v/transfer", accountID, domainName))
	transferResponse := &DomainTransferResponse{}

	// TODO: validate mandatory attributes RegistrantID

	resp, err := s.client.post(path, request, transferResponse)
	if err != nil {
		return nil, err
	}

	transferResponse.HttpResponse = resp
	return transferResponse, nil
}

// DomainTransferOutResponse represents a response from an API method that results in a domain transfer out.
type DomainTransferOutResponse struct {
	Response
	Data *Domain `json:"data"`
}

// Transfer out a domain name.
//
// See https://developer.dnsimple.com/v2/registrar/#transfer-out
func (s *RegistrarService) TransferDomainOut(accountID string, domainName string) (*DomainTransferOutResponse, error) {
	path := versioned(fmt.Sprintf("/%v/registrar/domains/%v/transfer_out", accountID, domainName))
	transferResponse := &DomainTransferOutResponse{}

	resp, err := s.client.post(path, nil, nil)
	if err != nil {
		return nil, err
	}

	transferResponse.HttpResponse = resp
	return transferResponse, nil
}

// DomainRenewRequest represents the attributes you can pass to a renew API request.
// Some attributes are mandatory.
type DomainRenewRequest struct {
	// The number of years
	Period int `json:"period"`
}

// DomainRenewal represents the result of a domain renewal call.
type DomainRenewal struct {
	ID           int    `json:"id"`
	DomainID     int    `json:"domain_id"`
	Period       int    `json:"period"`
	State        string `json:"state"`
	PremiumPrice string `json:"premium_price"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// DomainRenewalResponse represents a response from an API method that returns a domain renewal.
type DomainRenewalResponse struct {
	Response
	Data *DomainRenewal `json:"data"`
}

// RenewDomain renews a domain name.
//
// See https://developer.dnsimple.com/v2/registrar/#register
func (s *RegistrarService) RenewDomain(accountID string, domainName string, request *DomainRenewRequest) (*DomainRenewalResponse, error) {
	path := versioned(fmt.Sprintf("/%v/registrar/domains/%v/renewal", accountID, domainName))
	renewalResponse := &DomainRenewalResponse{}

	resp, err := s.client.post(path, request, renewalResponse)
	if err != nil {
		return nil, err
	}

	renewalResponse.HttpResponse = resp
	return renewalResponse, nil
}
