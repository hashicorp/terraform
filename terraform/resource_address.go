package terraform

import (
	"fmt"
	"regexp"
	"strconv"
)

// ResourceAddress is a way of identifying an individual resource (or,
// eventually, a subset of resources) within the state. It is used for Targets.
type ResourceAddress struct {
	Index        int
	InstanceType InstanceType
	Name         string
	Type         string
}

func ParseResourceAddress(s string) (*ResourceAddress, error) {
	matches, err := tokenizeResourceAddress(s)
	if err != nil {
		return nil, err
	}
	resourceIndex := -1
	if matches["index"] != "" {
		var err error
		if resourceIndex, err = strconv.Atoi(matches["index"]); err != nil {
			return nil, err
		}
	}
	instanceType := TypePrimary
	if matches["instance_type"] != "" {
		var err error
		if instanceType, err = ParseInstanceType(matches["instance_type"]); err != nil {
			return nil, err
		}
	}

	return &ResourceAddress{
		Index:        resourceIndex,
		InstanceType: instanceType,
		Name:         matches["name"],
		Type:         matches["type"],
	}, nil
}

func (addr *ResourceAddress) Equals(raw interface{}) bool {
	other, ok := raw.(*ResourceAddress)
	if !ok {
		return false
	}

	indexMatch := (addr.Index == -1 ||
		other.Index == -1 ||
		addr.Index == other.Index)

	return (indexMatch &&
		addr.InstanceType == other.InstanceType &&
		addr.Name == other.Name &&
		addr.Type == other.Type)
}

func ParseInstanceType(s string) (InstanceType, error) {
	switch s {
	case "primary":
		return TypePrimary, nil
	case "deposed":
		return TypeDeposed, nil
	case "tainted":
		return TypeTainted, nil
	default:
		return TypeInvalid, fmt.Errorf("Unexpected value for InstanceType field: %q", s)
	}
}

func tokenizeResourceAddress(s string) (map[string]string, error) {
	// Example of portions of the regexp below using the
	// string "aws_instance.web.tainted[1]"
	re := regexp.MustCompile(`\A` +
		// "aws_instance"
		`(?P<type>\w+)\.` +
		// "web"
		`(?P<name>\w+)` +
		// "tainted" (optional, omission implies: "primary")
		`(?:\.(?P<instance_type>\w+))?` +
		// "1" (optional, omission implies: "0")
		`(?:\[(?P<index>\d+)\])?` +
		`\z`)
	groupNames := re.SubexpNames()
	rawMatches := re.FindAllStringSubmatch(s, -1)
	if len(rawMatches) != 1 {
		return nil, fmt.Errorf("Problem parsing address: %q", s)
	}
	matches := make(map[string]string)
	for i, m := range rawMatches[0] {
		matches[groupNames[i]] = m
	}
	return matches, nil
}
