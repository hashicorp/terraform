/*
 *
 * gocommon - Go library to interact with the JoyentCloud
 *
 *
 * Copyright (c) 2016 Joyent Inc.
 *
 * Written by Daniele Stroppa <daniele.stroppa@joyent.com>
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package jpc

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"

	"github.com/joyent/gosign/auth"
)

const (
	// Environment variables
	SdcAccount = "SDC_ACCOUNT"
	SdcKeyId   = "SDC_KEY_ID"
	SdcUrl     = "SDC_URL"
	MantaUser  = "MANTA_USER"
	MantaKeyId = "MANTA_KEY_ID"
	MantaUrl   = "MANTA_URL"
)

var Locations = map[string]string{
	"us-east-1": "America/New_York",
	"us-west-1": "America/Los_Angeles",
	"us-sw-1":   "America/Los_Angeles",
	"eu-ams-1":  "Europe/Amsterdam",
}

// getConfig returns the value of the first available environment
// variable, among the given ones.
func getConfig(envVars ...string) (value string) {
	value = ""
	for _, v := range envVars {
		value = os.Getenv(v)
		if value != "" {
			break
		}
	}
	return
}

// getUserHome returns the value of HOME environment
// variable for the user environment.
func getUserHome() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA")
	} else {
		return os.Getenv("HOME")
	}
}

// credentialsFromEnv creates and initializes the credentials from the
// environment variables.
func credentialsFromEnv(key string) (*auth.Credentials, error) {
	var keyName string
	if key == "" {
		keyName = getUserHome() + "/.ssh/id_rsa"
	} else {
		keyName = key
	}
	privateKey, err := ioutil.ReadFile(keyName)
	if err != nil {
		return nil, err
	}
	authentication, err := auth.NewAuth(getConfig(SdcAccount, MantaUser), string(privateKey), "rsa-sha256")
	if err != nil {
		return nil, err
	}

	return &auth.Credentials{
		UserAuthentication: authentication,
		SdcKeyId:           getConfig(SdcKeyId),
		SdcEndpoint:        auth.Endpoint{URL: getConfig(SdcUrl)},
		MantaKeyId:         getConfig(MantaKeyId),
		MantaEndpoint:      auth.Endpoint{URL: getConfig(MantaUrl)},
	}, nil
}

// CompleteCredentialsFromEnv gets and verifies all the required
// authentication parameters have values in the environment.
func CompleteCredentialsFromEnv(keyName string) (cred *auth.Credentials, err error) {
	cred, err = credentialsFromEnv(keyName)
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(cred).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.String() == "" {
			return nil, fmt.Errorf("Required environment variable not set for credentials attribute: %s", t.Field(i).Name)
		}
	}
	return cred, nil
}
