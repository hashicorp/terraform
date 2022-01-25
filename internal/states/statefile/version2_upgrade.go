package statefile

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/copystructure"
)

func upgradeStateV2ToV3(old *stateV2) (*stateV3, error) {
	if old == nil {
		return (*stateV3)(nil), nil
	}

	var new *stateV3
	{
		copy, err := copystructure.Config{Lock: true}.Copy(old)
		if err != nil {
			panic(err)
		}
		newWrongType := copy.(*stateV2)
		newRightType := (stateV3)(*newWrongType)
		new = &newRightType
	}

	// Set the new version number
	new.Version = 3

	// Change the counts for things which look like maps to use the %
	// syntax. Remove counts for empty collections - they will be added
	// back in later.
	for _, module := range new.Modules {
		for _, resource := range module.Resources {
			// Upgrade Primary
			if resource.Primary != nil {
				upgradeAttributesV2ToV3(resource.Primary)
			}

			// Upgrade Deposed
			for _, deposed := range resource.Deposed {
				upgradeAttributesV2ToV3(deposed)
			}
		}
	}

	return new, nil
}

func upgradeAttributesV2ToV3(instanceState *instanceStateV2) error {
	collectionKeyRegexp := regexp.MustCompile(`^(.*\.)#$`)
	collectionSubkeyRegexp := regexp.MustCompile(`^([^\.]+)\..*`)

	// Identify the key prefix of anything which is a collection
	var collectionKeyPrefixes []string
	for key := range instanceState.Attributes {
		if submatches := collectionKeyRegexp.FindAllStringSubmatch(key, -1); len(submatches) > 0 {
			collectionKeyPrefixes = append(collectionKeyPrefixes, submatches[0][1])
		}
	}
	sort.Strings(collectionKeyPrefixes)

	log.Printf("[STATE UPGRADE] Detected the following collections in state: %v", collectionKeyPrefixes)

	// This could be rolled into fewer loops, but it is somewhat clearer this way, and will not
	// run very often.
	for _, prefix := range collectionKeyPrefixes {
		// First get the actual keys that belong to this prefix
		var potentialKeysMatching []string
		for key := range instanceState.Attributes {
			if strings.HasPrefix(key, prefix) {
				potentialKeysMatching = append(potentialKeysMatching, strings.TrimPrefix(key, prefix))
			}
		}
		sort.Strings(potentialKeysMatching)

		var actualKeysMatching []string
		for _, key := range potentialKeysMatching {
			if submatches := collectionSubkeyRegexp.FindAllStringSubmatch(key, -1); len(submatches) > 0 {
				actualKeysMatching = append(actualKeysMatching, submatches[0][1])
			} else {
				if key != "#" {
					actualKeysMatching = append(actualKeysMatching, key)
				}
			}
		}
		actualKeysMatching = uniqueSortedStrings(actualKeysMatching)

		// Now inspect the keys in order to determine whether this is most likely to be
		// a map, list or set. There is room for error here, so we log in each case. If
		// there is no method of telling, we remove the key from the InstanceState in
		// order that it will be recreated. Again, this could be rolled into fewer loops
		// but we prefer clarity.

		oldCountKey := fmt.Sprintf("%s#", prefix)

		// First, detect "obvious" maps - which have non-numeric keys (mostly).
		hasNonNumericKeys := false
		for _, key := range actualKeysMatching {
			if _, err := strconv.Atoi(key); err != nil {
				hasNonNumericKeys = true
			}
		}
		if hasNonNumericKeys {
			newCountKey := fmt.Sprintf("%s%%", prefix)

			instanceState.Attributes[newCountKey] = instanceState.Attributes[oldCountKey]
			delete(instanceState.Attributes, oldCountKey)
			log.Printf("[STATE UPGRADE] Detected %s as a map. Replaced count = %s",
				strings.TrimSuffix(prefix, "."), instanceState.Attributes[newCountKey])
		}

		// Now detect empty collections and remove them from state.
		if len(actualKeysMatching) == 0 {
			delete(instanceState.Attributes, oldCountKey)
			log.Printf("[STATE UPGRADE] Detected %s as an empty collection. Removed from state.",
				strings.TrimSuffix(prefix, "."))
		}
	}

	return nil
}

// uniqueSortedStrings removes duplicates from a slice of strings and returns
// a sorted slice of the unique strings.
func uniqueSortedStrings(input []string) []string {
	uniquemap := make(map[string]struct{})
	for _, str := range input {
		uniquemap[str] = struct{}{}
	}

	output := make([]string, len(uniquemap))

	i := 0
	for key := range uniquemap {
		output[i] = key
		i = i + 1
	}

	sort.Strings(output)
	return output
}
