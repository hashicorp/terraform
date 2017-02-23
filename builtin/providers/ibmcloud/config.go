package ibmcloud

import slsession "github.com/softlayer/softlayer-go/session"

//Config stores user provider input config and the API endpoints
type Config struct {
	//The IBM ID
	IBMID string
	//Password fo the IBM ID
	Password string
	//Bluemix region
	Region string
	//Bluemix API timeout
	Timeout string
	//Softlayer API key
	SoftLayerAPIKey string
	//Sofltayer user name
	SoftLayerUsername string
	//Softlayer end point url
	SoftLayerEndpointURL string
	//Softlayer API timeout
	SoftLayerTimeout string
	// Softlayer Account Number
	SoftLayerAccountNumber string

	//Bluemix API endpoint
	Endpoint string
	//IAM endpoint
	IAMEndpoint string
}

// ClientSession  contains  Bluemix and SoftLayer session
type ClientSession interface {
	SoftLayerSession() *slsession.Session
	BluemixSession() *Session
}

type clientSession struct {
	session *Session
}

// Method to provide the SoftLayer Session
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
