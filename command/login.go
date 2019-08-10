package command

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/command/cliconfig"
	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/tfdiags"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

// LoginCommand is a Command implementation that runs an interactive login
// flow for a remote service host. It then stashes credentials in a tfrc
// file in the user's home directory.
type LoginCommand struct {
	Meta
}

// Run implements cli.Command.
func (c *LoginCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.extendedFlagSet("login")
	var intoFile string
	cmdFlags.StringVar(&intoFile, "into-file", "", "set the file that the credentials will be appended to")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The login command expects at most one argument: the host to log in to.")
		cmdFlags.Usage()
		return 1
	}

	var diags tfdiags.Diagnostics

	if !c.input {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Login is an interactive command",
			"The \"terraform login\" command uses interactive prompts to obtain and record credentials, so it can't be run with input disabled.\n\nTo configure credentials in a non-interactive context, write existing credentials directly to a CLI configuration file.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	givenHostname := "app.terraform.io"
	if len(args) != 0 {
		givenHostname = args[0]
	}

	hostname, err := svchost.ForComparison(givenHostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid hostname",
			fmt.Sprintf("The given hostname %q is not valid: %s.", givenHostname, err.Error()),
		))
		c.showDiagnostics(diags)
		return 1
	}

	// From now on, since we've validated the given hostname, we should use
	// dispHostname in the UI to ensure we're presenting it in the canonical
	// form, in case that helpers users with debugging when things aren't
	// working as expected. (Perhaps the normalization is part of the cause.)
	dispHostname := hostname.ForDisplay()

	host, err := c.Services.Discover(hostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Service discovery failed for"+dispHostname,

			// Contrary to usual Go idiom, the Discover function returns
			// full sentences with initial capitalization in its error messages,
			// and they are written with the end-user as the audience. We
			// only need to add the trailing period to make them consistent
			// with our usual error reporting standards.
			err.Error()+".",
		))
		c.showDiagnostics(diags)
		return 1
	}

	creds := c.Services.CredentialsSource()

	// In normal use (i.e. without test mocks/fakes) creds will be an instance
	// of the command/cliconfig.CredentialsSource type, which has some extra
	// methods we can use to give the user better feedback about what we're
	// going to do. credsCtx will be nil if it's any other implementation,
	// though.
	var credsCtx *loginCredentialsContext
	if c, ok := creds.(*cliconfig.CredentialsSource); ok {
		filename, _ := c.CredentialsFilePath()
		credsCtx = &loginCredentialsContext{
			Location:      c.HostCredentialsLocation(hostname),
			LocalFilename: filename, // empty in the very unlikely event that we can't select a config directory for this user
			HelperType:    c.CredentialsHelperType(),
		}
	}

	clientConfig, err := host.ServiceOAuthClient("login.v1")
	switch err.(type) {
	case nil:
		// Great! No problem, then.
	case *disco.ErrServiceNotProvided:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Host does not support Terraform login",
			fmt.Sprintf("The given hostname %q does not allow creating Terraform authorization tokens.", dispHostname),
		))
	case *disco.ErrVersionNotSupported:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Host does not support Terraform login",
			fmt.Sprintf("The given hostname %q allows creating Terraform authorization tokens, but requires a newer version of Terraform CLI to do so.", dispHostname),
		))
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Host does not support Terraform login",
			fmt.Sprintf("The given hostname %q cannot support \"terraform login\": %s.", dispHostname, err),
		))
	}

	if credsCtx.Location == cliconfig.CredentialsInOtherFile {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Credentials for %s are manually configured", dispHostname),
			"The \"terraform login\" command cannot log in because credentials for this host are already configured in a CLI configuration file.\n\nTo log in, first revoke the existing credentials and remove that block from the CLI configuration.",
		))
	}

	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	var token *oauth2.Token
	switch {
	case clientConfig.SupportedGrantTypes.Has(disco.OAuthAuthzCodeGrant):
		// We prefer an OAuth code grant if the server supports it.
		var tokenDiags tfdiags.Diagnostics
		token, tokenDiags = c.interactiveGetTokenByCode(hostname, credsCtx, clientConfig)
		diags = diags.Append(tokenDiags)
		if tokenDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	case clientConfig.SupportedGrantTypes.Has(disco.OAuthOwnerPasswordGrant) && hostname == svchost.Hostname("app.terraform.io"):
		// The password grant type is allowed only for Terraform Cloud SaaS.
		var tokenDiags tfdiags.Diagnostics
		token, tokenDiags = c.interactiveGetTokenByPassword(hostname, credsCtx, clientConfig)
		diags = diags.Append(tokenDiags)
		if tokenDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Host does not support Terraform login",
			fmt.Sprintf("The given hostname %q does not allow any OAuth grant types that are supported by this version of Terraform.", dispHostname),
		))
		c.showDiagnostics(diags)
		return 1
	}

	// TODO: Save the token in the CLI config.
	// Also, if the token has an expiration time associated with it, prompt
	// the user that they will need to log in again after that time.
	fmt.Printf("Token is %#v\n", token)

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	return 0
}

// Help implements cli.Command.
func (c *LoginCommand) Help() string {
	defaultFile := c.defaultOutputFile()
	if defaultFile == "" {
		// Because this is just for the help message and it's very unlikely
		// that a user wouldn't have a functioning home directory anyway,
		// we'll just use a placeholder here. The real command has some
		// more complex behavior for this case. This result is not correct
		// on all platforms, but given how unlikely we are to hit this case
		// that seems okay.
		defaultFile = "~/.terraform/credentials.tfrc"
	}

	helpText := fmt.Sprintf(`
Usage: terraform login [options] [hostname]

  Retrieves an authentication token for the given hostname, if it supports
  automatic login, and saves it in a credentials file in your home directory.

  If no hostname is provided, the default hostname is app.terraform.io, to
  log in to Terraform Cloud.

  If not overridden by the -into-file option, the output file is:
      %s

Options:

  -into-file=....     Override which file the credentials block will be written
                      to. If this file already exists then it must have valid
                      HCL syntax and Terraform will update it in-place.
`, defaultFile)
	return strings.TrimSpace(helpText)
}

// Synopsis implements cli.Command.
func (c *LoginCommand) Synopsis() string {
	return "Obtain and save credentials for a remote host"
}

func (c *LoginCommand) defaultOutputFile() string {
	if c.CLIConfigDir == "" {
		return "" // no default available
	}
	return filepath.Join(c.CLIConfigDir, "credentials.tfrc")
}

func (c *LoginCommand) interactiveGetTokenByCode(hostname svchost.Hostname, credsCtx *loginCredentialsContext, clientConfig *disco.OAuthClient) (*oauth2.Token, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	confirm, confirmDiags := c.interactiveContextConsent(hostname, disco.OAuthAuthzCodeGrant, credsCtx)
	diags = diags.Append(confirmDiags)
	if !confirm {
		diags = diags.Append(errors.New("Login cancelled"))
		return nil, diags
	}

	// We'll use an entirely pseudo-random UUID for our temporary request
	// state. The OAuth server must echo this back to us in the callback
	// request to make it difficult for some other running process to
	// interfere by sending its own request to our temporary server.
	reqState, err := uuid.GenerateUUID()
	if err != nil {
		// This should be very unlikely, but could potentially occur if e.g.
		// there's not enough pseudo-random entropy available.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't generate login request state",
			fmt.Sprintf("Cannot generate random request identifier for login request: %s.", err),
		))
		return nil, diags
	}

	proofKey, proofKeyChallenge, err := c.proofKey()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't generate login request state",
			fmt.Sprintf("Cannot generate random prrof key for login request: %s.", err),
		))
		return nil, diags
	}

	listener, callbackURL, err := c.listenerForCallback(clientConfig.MinPort, clientConfig.MaxPort)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't start temporary login server",
			fmt.Sprintf(
				"The login process uses OAuth, which requires starting a temporary HTTP server on localhost. However, no TCP port numbers between %d and %d are available to create such a server.",
				clientConfig.MinPort, clientConfig.MaxPort,
			),
		))
		return nil, diags
	}

	// codeCh will allow our temporary HTTP server to transmit the OAuth code
	// to the main execution path that follows.
	codeCh := make(chan string)
	server := &http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			err := req.ParseForm()
			if err != nil {
				log.Printf("[ERROR] login: cannot ParseForm on callback request: %s", err)
				resp.WriteHeader(400)
				return
			}
			gotState := req.Form.Get("state")
			if gotState != reqState {
				log.Printf("[ERROR] login: incorrect \"state\" value in callback request")
				resp.WriteHeader(400)
				return
			}
			gotCode := req.Form.Get("code")
			if gotCode == "" {
				log.Printf("[ERROR] login: no \"code\" argument in callback request")
				resp.WriteHeader(400)
				return
			}

			// Send the code to our blocking wait below, so that the token
			// fetching process can continue.
			codeCh <- gotCode
			close(codeCh)

			resp.Header().Add("Content-Type", "text/html")
			resp.WriteHeader(200)
			resp.Write([]byte(callbackSuccessMessage))
		}),
	}
	go func() {
		err = server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Can't start temporary login server",
				fmt.Sprintf(
					"The login process uses OAuth, which requires starting a temporary HTTP server on localhost. However, no TCP port numbers between %d and %d are available to create such a server.",
					clientConfig.MinPort, clientConfig.MaxPort,
				),
			))
			close(codeCh)
		}
	}()

	oauthConfig := &oauth2.Config{
		ClientID:    clientConfig.ID,
		Endpoint:    clientConfig.Endpoint(),
		RedirectURL: callbackURL,
	}

	authCodeURL := oauthConfig.AuthCodeURL(
		reqState,
		oauth2.SetAuthURLParam("code_challenge", proofKeyChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	err = browser.OpenURL(authCodeURL)
	if err == nil {
		c.Ui.Output(fmt.Sprintf("Terraform must now open a web browser to the login page for %s.\n", hostname.ForDisplay()))
		c.Ui.Output(fmt.Sprintf("If a browser does not open this automatically, open the following URL to proceed:\n    %s\n", authCodeURL))
	} else {
		// Assume we're on a platform where opening a browser isn't possible.
		c.Ui.Output(fmt.Sprintf("Open the following URL to access the login page for %s:\n    %s\n", hostname.ForDisplay(), authCodeURL))
	}

	c.Ui.Output("Terraform will now wait for the host to signal that login was successful.\n")

	code, ok := <-codeCh
	if !ok {
		// If we got no code at all then the server wasn't able to start
		// up, so we'll just give up.
		return nil, diags
	}

	err = server.Close()
	if err != nil {
		// The server will close soon enough when our process exits anyway,
		// so we won't fuss about it for right now.
		log.Printf("[WARN] login: callback server can't shut down: %s", err)
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpclient.New())
	token, err := oauthConfig.Exchange(
		ctx, code,
		oauth2.SetAuthURLParam("code_verifier", proofKey),
	)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to obtain auth token",
			fmt.Sprintf("The remote server did not assign an auth token: %s.", err),
		))
		return nil, diags
	}

	return token, diags
}

func (c *LoginCommand) interactiveGetTokenByPassword(hostname svchost.Hostname, credsCtx *loginCredentialsContext, clientConfig *disco.OAuthClient) (*oauth2.Token, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	confirm, confirmDiags := c.interactiveContextConsent(hostname, disco.OAuthOwnerPasswordGrant, credsCtx)
	diags = diags.Append(confirmDiags)
	if !confirm {
		diags = diags.Append(errors.New("Login cancelled"))
		return nil, diags
	}

	return nil, diags
}

func (c *LoginCommand) interactiveContextConsent(hostname svchost.Hostname, grantType disco.OAuthGrantType, credsCtx *loginCredentialsContext) (bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	c.Ui.Output(fmt.Sprintf("Terraform will request an API token for %s using OAuth.\n", hostname.ForDisplay()))

	if grantType.UsesAuthorizationEndpoint() {
		c.Ui.Output(
			"This will work only if you are able to use a web browser on this computer to\ncomplete a login process. If not, you must obtain an API token by another\nmeans and configure it in the CLI configuration manually.\n",
		)
	}

	// credsCtx might not be set if we're using a mock credentials source
	// in a test, but it should always be set in normal use.
	if credsCtx != nil {
		switch credsCtx.Location {
		case cliconfig.CredentialsViaHelper:
			c.Ui.Output(fmt.Sprintf("If login is successful, Terraform will store the token in the configured\n%q credentials helper for use by subsequent commands.\n", credsCtx.HelperType))
		case cliconfig.CredentialsInPrimaryFile, cliconfig.CredentialsNotAvailable:
			c.Ui.Output(fmt.Sprintf("If login is successful, Terraform will store the token in plain text in\nthe following file for use by subsequent commands:\n    %s\n", credsCtx.LocalFilename))
		}
	}

	v, err := c.prompt("Do you want to proceed? (y/n)", false)
	if err != nil {
		// Should not happen because this command checks that input is enabled
		// before we get to this point.
		diags = diags.Append(err)
		return false, diags
	}

	switch strings.ToLower(v) {
	case "y", "yes":
		return true, diags
	default:
		return false, diags
	}
}

func (c *LoginCommand) listenerForCallback(minPort, maxPort uint16) (net.Listener, string, error) {
	if minPort < 1024 || maxPort < 1024 {
		// This should never happen because it should've been checked by
		// the svchost/disco package when reading the service description,
		// but we'll prefer to fail hard rather than inadvertently trying
		// to open an unprivileged port if there are bugs at that layer.
		panic("listenerForCallback called with privileged port number")
	}

	availCount := int(maxPort) - int(minPort)

	// We're going to try port numbers within the range at random, so we need
	// to terminate eventually in case _none_ of the ports are available.
	// We'll make that 150% of the number of ports just to give us some room
	// for the random number generator to generate the same port more than
	// once.
	// Note that we don't really care about true randomness here... we're just
	// trying to hop around in the available port space rather than always
	// working up from the lowest, because we have no information to predict
	// that any particular number will be more likely to be available than
	// another.
	maxTries := availCount + (availCount / 2)

	for tries := 0; tries < maxTries; tries++ {
		port := rand.Intn(availCount) + int(minPort)
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		log.Printf("[TRACE] login: trying %s as a listen address for temporary OAuth callback server", addr)
		l, err := net.Listen("tcp4", addr)
		if err == nil {
			// We use a path that doesn't end in a slash here because some
			// OAuth server implementations don't allow callback URLs to
			// end with slashes.
			callbackURL := fmt.Sprintf("http://localhost:%d/login", port)
			log.Printf("[TRACE] login: callback URL will be %s", callbackURL)
			return l, callbackURL, nil
		}
	}

	return nil, "", fmt.Errorf("no suitable TCP ports (between %d and %d) are available for the temporary OAuth callback server", minPort, maxPort)
}

func (c *LoginCommand) proofKey() (key, challenge string, err error) {
	// Wel use a UUID-like string as the "proof key for code exchange" (PKCE)
	// that will eventually authenticate our request to the token endpoint.
	// Standard UUIDs are explicitly not suitable as secrets according to the
	// UUID spec, but our go-uuid just generates totally random number sequences
	// formatted in the conventional UUID syntax, so that concern does not
	// apply here: this is just a 128-bit crypto-random number.
	key, err = uuid.GenerateUUID()
	if err != nil {
		return "", "", err
	}

	h := sha256.New()
	h.Write([]byte(key))
	challenge = base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return key, challenge, nil
}

type loginCredentialsContext struct {
	Location      cliconfig.CredentialsLocation
	LocalFilename string
	HelperType    string
}

const callbackSuccessMessage = `
<html>
<head>
<title>Terraform Login</title>
<style type="text/css">
body {
	font-family: monospace;
	color: #fff;
	background-color: #000;
}
</style>
</head>
<body>

<p>The login server has returned an authentication code to Terraform.</p>
<p>Now close this page and return to the terminal where <tt>terraform login</tt>
is running to see the result of the login process.</p>

</body>
</html>
`
