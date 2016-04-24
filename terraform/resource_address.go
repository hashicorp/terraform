package terraform

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ResourceAddress is a way of identifying an individual resource (or,
// eventually, a subset of resources) within the state. It is used for Targets.
type ResourceAddress struct {
	// Addresses a resource falling somewhere in the module path
	// When specified alone, addresses all resources within a module path
	Path []string

	// Addresses a specific resource that occurs in a list
	Index int

	InstanceType InstanceType
	Name         string
	Type         string
}

// Copy returns a copy of this ResourceAddress
func (r *ResourceAddress) Copy() *ResourceAddress {
	n := &ResourceAddress{
		Path:         make([]string, 0, len(r.Path)),
		Index:        r.Index,
		InstanceType: r.InstanceType,
		Name:         r.Name,
		Type:         r.Type,
	}
	for _, p := range r.Path {
		n.Path = append(n.Path, p)
	}
	return n
}

func ParseResourceAddress(s string) (*ResourceAddress, error) {
	matches, err := tokenizeResourceAddress(s)
	if err != nil {
		return nil, err
	}
	resourceIndex, err := ParseResourceIndex(matches["index"])
	if err != nil {
		return nil, err
	}
	instanceType, err := ParseInstanceType(matches["instance_type"])
	if err != nil {
		return nil, err
	}
	path := ParseResourcePath(matches["path"])

	return &ResourceAddress{
		Path:         path,
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

	pathMatch := len(addr.Path) == 0 && len(other.Path) == 0 ||
		reflect.DeepEqual(addr.Path, other.Path)

	indexMatch := addr.Index == -1 ||
		other.Index == -1 ||
		addr.Index == other.Index

	nameMatch := addr.Name == "" ||
		other.Name == "" ||
		addr.Name == other.Name

	typeMatch := addr.Type == "" ||
		other.Type == "" ||
		addr.Type == other.Type

	return pathMatch &&
		indexMatch &&
		addr.InstanceType == other.InstanceType &&
		nameMatch &&
		typeMatch
}

func ParseResourceIndex(s string) (int, error) {
	if s == "" {
		return -1, nil
	}
	return strconv.Atoi(s)
}

func ParseResourcePath(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ".")
	path := make([]string, 0, len(parts))
	for _, s := range parts {
		// Due to the limitations of the regexp match below, the path match has
		// some noise in it we have to filter out :|
		if s == "" || s == "module" {
			continue
		}
		path = append(path, s)
	}
	return path
}

func ParseInstanceType(s string) (InstanceType, error) {
	switch s {
	case "", "primary":
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
		// "module.foo.module.bar" (optional)
		`(?P<path>(?:module\.[^.]+\.?)*)` +
		// "aws_instance.web" (optional when module path specified)
		`(?:(?P<type>[^.]+)\.(?P<name>[^.[]+))?` +
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
