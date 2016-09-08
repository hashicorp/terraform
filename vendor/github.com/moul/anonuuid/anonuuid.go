package anonuuid

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	// UUIDRegex is the regex used to find UUIDs in texts
	UUIDRegex = "[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}"
)

// AnonUUID is the main structure, it contains the cache map and helpers
type AnonUUID struct {
	cache map[string]string

	guard sync.Mutex // cache guard

	// Hexspeak flag will generate hexspeak style fake UUIDs
	Hexspeak bool

	// Random flag will generate random fake UUIDs
	Random bool

	// Prefix will be the beginning of all the generated UUIDs
	Prefix string

	// Suffix will be the end of all the generated UUIDs
	Suffix string

	// AllowNonUUIDInput tells FakeUUID to accept non UUID input string
	AllowNonUUIDInput bool

	// KeepBeginning tells FakeUUID to let the beginning of the UUID as it is
	KeepBeginning bool

	// KeepEnd tells FakeUUID to let the last part of the UUID as it is
	KeepEnd bool
}

// Sanitize takes a string as input and return sanitized string
func (a *AnonUUID) Sanitize(input string) string {
	r := regexp.MustCompile(UUIDRegex)

	return r.ReplaceAllStringFunc(input, func(m string) string {
		parts := r.FindStringSubmatch(m)
		return a.FakeUUID(parts[0])
	})
}

// FakeUUID takes a word (real UUID or standard string) and returns its corresponding (mapped) fakeUUID
func (a *AnonUUID) FakeUUID(input string) string {
	if !a.AllowNonUUIDInput {
		err := IsUUID(input)
		if err != nil {
			return "invaliduuid"
		}
	}
	a.guard.Lock()
	defer a.guard.Unlock()
	if _, ok := a.cache[input]; !ok {

		if a.KeepBeginning {
			a.Prefix = input[:8]
		}

		if a.KeepEnd {
			a.Suffix = input[36-12:]
		}

		if a.Prefix != "" {
			matched, err := regexp.MatchString("^[a-z0-9]+$", a.Prefix)
			if err != nil || !matched {
				a.Prefix = "invalidprefix"
			}
		}

		if a.Suffix != "" {
			matched, err := regexp.MatchString("^[a-z0-9]+$", a.Suffix)
			if err != nil || !matched {
				a.Suffix = "invalsuffix"
			}
		}

		var fakeUUID string
		var err error
		if a.Hexspeak {
			fakeUUID, err = GenerateHexspeakUUID(len(a.cache))
		} else if a.Random {
			fakeUUID, err = GenerateRandomUUID(10)
		} else {
			fakeUUID, err = GenerateLenUUID(len(a.cache))
		}
		if err != nil {
			log.Fatalf("Failed to generate an UUID: %v", err)
		}

		if a.Prefix != "" {
			fakeUUID, err = PrefixUUID(a.Prefix, fakeUUID)
			if err != nil {
				panic(err)
			}
		}

		if a.Suffix != "" {
			fakeUUID, err = SuffixUUID(a.Suffix, fakeUUID)
			if err != nil {
				panic(err)
			}
		}

		// FIXME: check for duplicates and retry

		a.cache[input] = fakeUUID
	}
	return a.cache[input]
}

// New returns a prepared AnonUUID structure
func New() *AnonUUID {
	return &AnonUUID{
		cache:    make(map[string]string),
		Hexspeak: false,
		Random:   false,
	}
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// PrefixUUID returns a prefixed UUID
func PrefixUUID(prefix string, uuid string) (string, error) {
	uuidLetters := uuid[:8] + uuid[9:13] + uuid[14:18] + uuid[19:23] + uuid[24:36]
	prefixedUUID, err := FormatUUID(prefix + uuidLetters)
	if err != nil {
		return "", err
	}
	return prefixedUUID, nil
}

// SuffixUUID returns a suffixed UUID
func SuffixUUID(suffix string, uuid string) (string, error) {
	uuidLetters := uuid[:8] + uuid[9:13] + uuid[14:18] + uuid[19:23] + uuid[24:36]
	uuidLetters = uuidLetters[:32-len(suffix)] + suffix
	suffixedUUID, err := FormatUUID(uuidLetters)
	if err != nil {
		return "", err
	}
	return suffixedUUID, nil
}

// IsUUID returns nil if the input is an UUID, else it returns an error
func IsUUID(input string) error {
	matched, err := regexp.MatchString("^"+UUIDRegex+"$", input)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("String '%s' is not a valid UUID", input)
	}
	return nil
}

// FormatUUID takes a string in input and return an UUID formatted string by repeating the string and placing dashes if necessary
func FormatUUID(part string) (string, error) {
	if len(part) < 1 {
		return "", fmt.Errorf("Empty UUID")
	}
	if len(part) < 32 {
		part = strings.Repeat(part, 32)
	}
	if len(part) > 32 {
		part = part[:32]
	}
	uuid := part[:8] + "-" + part[8:12] + "-1" + part[13:16] + "-" + part[16:20] + "-" + part[20:32]

	err := IsUUID(uuid)
	if err != nil {
		return "", err
	}

	return uuid, nil
}

// GenerateRandomUUID returns an UUID based on random strings
func GenerateRandomUUID(length int) (string, error) {
	var letters = []rune("abcdef0123456789")

	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return FormatUUID(string(b))
}

// GenerateHexspeakUUID returns an UUID formatted string containing hexspeak words
func GenerateHexspeakUUID(i int) (string, error) {
	if i < 0 {
		i = -i
	}
	hexspeaks := []string{
		"0ff1ce",
		"31337",
		"4b1d",
		"badc0de",
		"badcafe",
		"badf00d",
		"deadbabe",
		"deadbeef",
		"deadc0de",
		"deadfeed",
		"fee1bad",
	}
	return FormatUUID(hexspeaks[i%len(hexspeaks)])
}

// GenerateLenUUID returns an UUID formatted string based on an index number
func GenerateLenUUID(i int) (string, error) {
	if i < 0 {
		i = 2<<29 + i
	}
	return FormatUUID(fmt.Sprintf("%x", i))
}
