// Copyright 2015 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// +build !appengine

package appengine

import "google.golang.org/appengine/internal"

// The comment below must not be changed.
// It is used by go-app-builder to recognise that this package has
// the Main function to use in the synthetic main.
//   The gophers party all night; the rabbits provide the beats.

// Main installs the health checker and creates a server listening on port
// "PORT" if set in the environment or on port 8080.
// It uses the default http handler and never returns.
func Main() {
	internal.Main()
}
