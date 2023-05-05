// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"testing"
)

func TestMap(t *testing.T) {
	variableName := InputVariable{Name: "name"}
	localHello := LocalValue{Name: "hello"}
	pathModule := PathAttr{Name: "module"}
	moduleBeep := ModuleCall{Name: "beep"}
	eachKey := ForEachAttr{Name: "key"} // intentionally not in the map

	m := MakeMap(
		MakeMapElem[Referenceable](variableName, "Aisling"),
	)

	m.Put(localHello, "hello")
	m.Put(pathModule, "boop")
	m.Put(moduleBeep, "unrealistic")

	keySet := m.Keys()
	if want := variableName; !m.Has(want) {
		t.Errorf("map does not include %s", want)
	}
	if want := variableName; !keySet.Has(want) {
		t.Errorf("key set does not include %s", want)
	}
	if want := localHello; !m.Has(want) {
		t.Errorf("map does not include %s", want)
	}
	if want := localHello; !keySet.Has(want) {
		t.Errorf("key set does not include %s", want)
	}
	if want := pathModule; !keySet.Has(want) {
		t.Errorf("key set does not include %s", want)
	}
	if want := moduleBeep; !keySet.Has(want) {
		t.Errorf("key set does not include %s", want)
	}
	if doNotWant := eachKey; m.Has(doNotWant) {
		t.Errorf("map includes rogue element %s", doNotWant)
	}
	if doNotWant := eachKey; keySet.Has(doNotWant) {
		t.Errorf("key set includes rogue element %s", doNotWant)
	}

	if got, want := m.Get(variableName), "Aisling"; got != want {
		t.Errorf("unexpected value %q for %s; want %q", got, variableName, want)
	}
	if got, want := m.Get(localHello), "hello"; got != want {
		t.Errorf("unexpected value %q for %s; want %q", got, localHello, want)
	}
	if got, want := m.Get(pathModule), "boop"; got != want {
		t.Errorf("unexpected value %q for %s; want %q", got, pathModule, want)
	}
	if got, want := m.Get(moduleBeep), "unrealistic"; got != want {
		t.Errorf("unexpected value %q for %s; want %q", got, moduleBeep, want)
	}
	if got, want := m.Get(eachKey), ""; got != want {
		// eachKey isn't in the map, so Get returns the zero value of string
		t.Errorf("unexpected value %q for %s; want %q", got, eachKey, want)
	}

	if v, ok := m.GetOk(variableName); v != "Aisling" || !ok {
		t.Errorf("GetOk for %q returned incorrect result (%q, %#v)", variableName, v, ok)
	}
	if v, ok := m.GetOk(eachKey); v != "" || ok {
		t.Errorf("GetOk for %q returned incorrect result (%q, %#v)", eachKey, v, ok)
	}

	m.Remove(moduleBeep)
	if doNotWant := moduleBeep; m.Has(doNotWant) {
		t.Errorf("map still includes %s after removing it", doNotWant)
	}
	if want := moduleBeep; !keySet.Has(want) {
		t.Errorf("key set no longer includes %s after removing it from the map; key set is supposed to be a snapshot at the time of call", want)
	}
	keySet = m.Keys()
	if doNotWant := moduleBeep; keySet.Has(doNotWant) {
		t.Errorf("key set still includes %s after a second call after removing it from the map", doNotWant)
	}
}
