package ibmcloud

import (
	"time"

	slsession "github.com/softlayer/softlayer-go/session"
)

//Config stores user provider input config and the API endpoints
type Config struct {
	//The IBM ID
	IBMID string
	//Password fo the IBM ID
	IBMIDPassword string
	//Bluemix region
	Region string
	//Softlayer end point url
	SoftLayerEndpointURL string
	//SoftlayerXMLRPCEndpoint endpoint
	SoftlayerXMLRPCEndpoint string
	//Softlayer API timeout
	SoftLayerTimeout time.Duration
	// Softlayer Account Number
	SoftLayerAccountNumber string

	//IAM endpoint
	IAMEndpoint string

	//Retry Count for API calls
	//Unexposed in the schema at this point as they are used only during session creation for a few calls
	//When sdk implements it we an expose them for expected behaviour
	//https://github.com/softlayer/softlayer-go/issues/41
	RetryCount int
	//Constant Retry Delay for API calls
	RetryDelay time.Duration
}

// ClientSession  contains  Bluemix and SoftLayer session
type ClientSession interface {
	SoftLayerSession() *slsession.Session
	BluemixSession() *Session
}

//clientSession implements the ClientSession interface
type clientSession struct {
	session *Session
}

// SoftLayerSession providers SoftLayer Session
func (sess clientSession) SoftLayerSession() *slsession.Session {
	return sess.session.SoftLayerSession
}

// Method to provide the Bluemix Session
func (sess clientSession) BluemixSession() *Session {
	return sess.session
}

// ClientSession configures and returns a fully initialized ClientSession
func (c *Config) ClientSession() (interface{}, error) {
	sess, err := newSession(c)
	return clientSession{session: sess}, err
}
