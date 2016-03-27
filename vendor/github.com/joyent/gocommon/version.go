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

package gocommon

import (
	"fmt"
)

type VersionNum struct {
	Major int
	Minor int
	Micro int
}

func (v *VersionNum) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Micro)
}

var VersionNumber = VersionNum{
	Major: 0,
	Minor: 1,
	Micro: 0,
}

var Version = VersionNumber.String()
