// Package stressaddr contains some types that represent the "addresses" of
// various generated objects.
//
// These are typically just thin wrappers around random seed values, but we
// use a particular syntax for them to make it harder to e.g. confuse a
// series id with a config id when debugging, and end up trying to reproduce
// an error against the wrong input.
//
// A generated object id remains valid only as long as it's being passed to
// exactly the same code that originally generated it, including both code
// within the stresstest packages and code in dependencies such as the Go
// standard library's random number generator. We therefore expect to use these
// only briefly as part of re-running a particular test case for debugging
// purposes; it's not useful to record these addresses in any long-term
// location unless you also retain the stresstest binary that originally
// issued them. However, they will be preserved under changes to other parts
// of Terraform that we run the generated configurations against, in case
// you want to add extra debug logs (or similar) to try to understand better
// the cause of a failure.
package stressaddr

// Internal note: math/rand typically uses int64 rather than uint64 for seeds,
// but rand.Rand doesn't have a method specifically for generating int64 values,
// and unsigned integers make for (subjectively) nicer string representations
// of addresses anyway. For that reason, we use uint64 as our main type for
// addresses here and then do a reinterpret-cast to int64 in order to produce
// an actual seed.

import (
	"fmt"
	"math/rand"
	"strings"
)

// All of the "primitive" addresses have a string representation of 17
// bytes: the type discriminator letter followed by 16 hex digits.
const primAddrStrLen = 17

// Config is an address of either an initial configuration or a modification
// of that configuration. Config is built from a StartConfig representing the
// initial configuration and then zero or more ModConfig addresses representing
// a series of modifications made to the predecessor.
type Config struct {
	Start StartConfig
	Mods  []ModConfig
}

// FixedConfig constructs a new Config address with a fixed start configuration
// and zero or more modifications.
func FixedConfig(start StartConfig, mods ...ModConfig) Config {
	return Config{
		Start: start,
		Mods:  mods,
	}
}

// RandomConfig constructs a new Config address with a randomly-selected
// start configuration and a randomly-chosen number of modifications, with
// between zero and three randomly-chosen modification addresses.
func RandomConfig(rnd *rand.Rand) Config {
	start := RandomStartConfig(rnd)
	modCount := rnd.Intn(4)
	ret := Config{Start: start}
	if modCount != 0 {
		ret.Mods = make([]ModConfig, modCount)
	}
	for i := range ret.Mods {
		ret.Mods[i] = RandomModConfig(rnd)
	}
	return ret
}

// ParseConfig attempts to parse the given string as the string representation
// of a Config address, return the resulting value if successful.
func ParseConfig(raw string) (Config, error) {
	// The string representation of a Config is just the string concatenation
	// of all of its component parts, so a valid string's length always a
	// multiple of primAddrStrLen and each chunk should be a valid address
	// once isolated.
	if len(raw) == 0 || (len(raw)%primAddrStrLen) != 0 {
		return Config{}, fmt.Errorf("must be an initial config identifier followed by zero or more modification identifiers")
	}
	startRaw, raw := raw[:primAddrStrLen], raw[primAddrStrLen:]
	start, err := ParseStartConfig(startRaw)
	if err != nil {
		return Config{}, err
	}
	ret := Config{
		Start: start,
	}
	for len(raw) > 0 {
		var modRaw string
		modRaw, raw = raw[:primAddrStrLen], raw[primAddrStrLen:]
		mod, err := ParseModConfig(modRaw)
		if err != nil {
			return Config{}, err
		}
		ret.Mods = append(ret.Mods, mod)
	}
	return ret, nil
}

func (c Config) String() string {
	if len(c.Mods) == 0 {
		return c.Start.String() // Easy case
	}
	var b strings.Builder
	b.Grow(primAddrStrLen * (1 + len(c.Mods)))
	b.WriteString(c.Start.String())
	for _, mod := range c.Mods {
		b.WriteString(mod.String())
	}
	return b.String()
}

// NewMod constructs a new Config which matches the receiver except for having
// one additional modification appended to the end of it.
func (c Config) NewMod(addr ModConfig) Config {
	ret := Config{
		Start: c.Start,
		Mods:  make([]ModConfig, len(c.Mods), len(c.Mods)+1),
	}
	copy(ret.Mods, c.Mods)
	ret.Mods = append(ret.Mods, addr)
	return ret
}

// StartConfig is an address of a randomly-generated initial configuration.
type StartConfig uint64

const startConfigFmt = "C%016x"

// FixedStartConfig constructs a new StartConfig address with a fixed seed.
func FixedStartConfig(seed int64) StartConfig {
	return StartConfig(uint64(seed))
}

// RandomStartConfig uses the given random number generator to choose
// a random StartConfig address.
func RandomStartConfig(rnd *rand.Rand) StartConfig {
	return StartConfig(rnd.Uint64())
}

// ParseStartConfig attempts to parse the given string as the string representation
// of a StartConfig address, returning the resulting value if successful.
func ParseStartConfig(raw string) (StartConfig, error) {
	var v uint64
	_, err := fmt.Sscanf(raw, startConfigFmt, &v)
	if err != nil || len(raw) != primAddrStrLen {
		return 0, fmt.Errorf("%q is not a valid initial configuration identifier", raw)
	}
	return StartConfig(v), nil
}

func (c StartConfig) String() string {
	return fmt.Sprintf(startConfigFmt, uint64(c))
}

// RandomSeed returns a value suitable for seeding a pseudorandom source in
// order to produce the configuration this address represents.
func (c StartConfig) RandomSeed() int64 {
	return int64(c)
}

// ModConfig is an address of a randomly-generated configuration modification.
//
// Note that ModConfig represents the modification itself, rather than the
// configuration that results from it: a ModConfig is only meaningful in
// the context of a particular predecessor configuration. The Config address
// type combines StartConfig and ModConfig together to produce fully-qualified
// addresses for potentially-modified configurations, so that's the more
// useful representation for most situations, unless you're writing code
// inside the config series generator itself.
type ModConfig uint64

const modConfigFmt = "M%016x"

// FixedModConfig constructs a new ModConfig address with a fixed seed.
func FixedModConfig(seed int64) ModConfig {
	return ModConfig(uint64(seed))
}

// RandomModConfig uses the given random number generator to choose
// a random ModConfig address.
func RandomModConfig(rnd *rand.Rand) ModConfig {
	return ModConfig(rnd.Uint64())
}

// ParseModConfig attempts to parse the given string as the string representation
// of a ModConfig address, returning the resulting value if successful.
func ParseModConfig(raw string) (ModConfig, error) {
	var v uint64
	_, err := fmt.Sscanf(raw, modConfigFmt, &v)
	if err != nil || len(raw) != primAddrStrLen {
		return 0, fmt.Errorf("%q is not a valid configuration modification identifier", raw)
	}
	return ModConfig(v), nil
}

func (c ModConfig) String() string {
	return fmt.Sprintf(modConfigFmt, uint64(c))
}

// RandomSeed returns a value suitable for seeding a pseudorandom source in
// order to produce the configuration this address represents.
func (c ModConfig) RandomSeed() int64 {
	return int64(c)
}

// ConfigSeries is an address of a randomly-generated configuration series.
type ConfigSeries uint64

const configSeriesFmt = "S%016x"

// FixedConfigSeries constructs a new ConfigSeries address with a fixed seed.
func FixedConfigSeries(seed int64) ConfigSeries {
	return ConfigSeries(uint64(seed))
}

// RandomConfigSeries uses the given random number generator to choose
// a random ConfigSeries address.
func RandomConfigSeries(rnd *rand.Rand) ConfigSeries {
	return ConfigSeries(rnd.Uint64())
}

// ParseConfigSeries attempts to parse the given string as the string
// representation of a ConfigSeries address, returning the resulting value if
// successful.
func ParseConfigSeries(raw string) (ConfigSeries, error) {
	var v uint64
	_, err := fmt.Sscanf(raw, configSeriesFmt, &v)
	if err != nil || len(raw) != primAddrStrLen {
		return 0, fmt.Errorf("%q is not a valid configuration series identifier", raw)
	}
	return ConfigSeries(v), nil
}

func (c ConfigSeries) String() string {
	return fmt.Sprintf(configSeriesFmt, uint64(c))
}

// RandomSeed returns a value suitable for seeding a pseudorandom source in
// order to produce the configuration series this address represents.
func (c ConfigSeries) RandomSeed() int64 {
	return int64(c)
}
