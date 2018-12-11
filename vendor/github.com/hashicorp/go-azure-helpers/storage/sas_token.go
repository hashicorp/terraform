package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

const (
	connStringAccountKeyKey  = "AccountKey"
	connStringAccountNameKey = "AccountName"
)

// ComputeAccountSASToken computes the SAS Token for a Storage Account based on the
// access key & given permissions
func ComputeAccountSASToken(accountName string,
	accountKey string,
	permissions string,
	services string,
	resourceTypes string,
	start string,
	expiry string,
	signedProtocol string,
	signedIp string, // nolint: unparam
	signedVersion string, // nolint: unparam
) (string, error) {

	// UTF-8 by default...
	stringToSign := accountName + "\n"
	stringToSign += permissions + "\n"
	stringToSign += services + "\n"
	stringToSign += resourceTypes + "\n"
	stringToSign += start + "\n"
	stringToSign += expiry + "\n"
	stringToSign += signedIp + "\n"
	stringToSign += signedProtocol + "\n"
	stringToSign += signedVersion + "\n"

	binaryKey, err := base64.StdEncoding.DecodeString(accountKey)
	if err != nil {
		return "", err
	}
	hasher := hmac.New(sha256.New, binaryKey)
	hasher.Write([]byte(stringToSign))
	signature := hasher.Sum(nil)

	// Trial and error to determine which fields the Azure portal
	// URL encodes for a query string and which it does not.
	sasToken := "?sv=" + url.QueryEscape(signedVersion)
	sasToken += "&ss=" + url.QueryEscape(services)
	sasToken += "&srt=" + url.QueryEscape(resourceTypes)
	sasToken += "&sp=" + url.QueryEscape(permissions)
	sasToken += "&se=" + (expiry)
	sasToken += "&st=" + (start)
	sasToken += "&spr=" + (signedProtocol)

	// this is consistent with how the Azure portal builds these.
	if len(signedIp) > 0 {
		sasToken += "&sip=" + signedIp
	}

	sasToken += "&sig=" + url.QueryEscape(base64.StdEncoding.EncodeToString(signature))

	return sasToken, nil
}

// ParseAccountSASConnectionString parses the Connection String for a Storage Account
func ParseAccountSASConnectionString(connString string) (map[string]string, error) {
	// This connection string was for a real storage account which has been deleted
	// so its safe to include here for reference to understand the format.
	// DefaultEndpointsProtocol=https;AccountName=azurermtestsa0;AccountKey=2vJrjEyL4re2nxCEg590wJUUC7PiqqrDHjAN5RU304FNUQieiEwS2bfp83O0v28iSfWjvYhkGmjYQAdd9x+6nw==;EndpointSuffix=core.windows.net
	validKeys := map[string]bool{"DefaultEndpointsProtocol": true, "BlobEndpoint": true,
		"AccountName": true, "AccountKey": true, "EndpointSuffix": true}
	// The k-v pairs are separated with semi-colons
	tokens := strings.Split(connString, ";")

	kvp := make(map[string]string)

	for _, atoken := range tokens {
		// The individual k-v are separated by an equals sign.
		kv := strings.SplitN(atoken, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("[ERROR] token `%s` is an invalid key=pair (connection string %s)", atoken, connString)
		}

		key := kv[0]
		val := kv[1]

		if _, present := validKeys[key]; !present {
			return nil, fmt.Errorf("[ERROR] Unknown Key `%s` in connection string %s", key, connString)
		}
		kvp[key] = val
	}

	if _, present := kvp[connStringAccountKeyKey]; !present {
		return nil, fmt.Errorf("[ERROR] Storage Account Key not found in connection string: %s", connString)
	}

	return kvp, nil
}
