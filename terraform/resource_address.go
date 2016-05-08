package terraform

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// ResourceAddress is a way of identifying an individual resource (or,
// eventually, a subset of resources) within the state. It is used for Targets.
type ResourceAddress struct {
	// Addresses a resource falling somewhere in the module path
	// When specified alone, addresses all resources within a module path
	Path []string

	// Addresses a specific resource that occurs in a list
	Index int

	InstanceType    InstanceType
	InstanceTypeSet bool
	Name            string
	Type            string
	Mode            config.ResourceMode // significant only if InstanceTypeSet
}

// Copy returns a copy of this ResourceAddress
func (r *ResourceAddress) Copy() *ResourceAddress {
	n := &ResourceAddress{
		Path:         make([]string, 0, len(r.Path)),
		Index:        r.Index,
		InstanceType: r.InstanceType,
		Name:         r.Name,
		Type:         r.Type,
		Mode:         r.Mode,
	}
	for _, p := range r.Path {
		n.Path = append(n.Path, p)
	}
	return n
}

// String outputs the address that parses into this address.
func (r *ResourceAddress) String() string {
	var result []string
	for _, p := range r.Path {
		result = append(result, "module", p)
	}

	switch r.Mode {
	case config.ManagedResourceMode:
		// nothing to do
	case config.DataResourceMode:
		result = append(result, "data")
	default:
		panic(fmt.Errorf("unsupported resource mode %s", r.Mode))
	}

	if r.Type != "" {
		result = append(result, r.Type)
	}

	if r.Name != "" {
		name := r.Name
		if r.InstanceTypeSet {
			switch r.InstanceType {
			case TypePrimary:
				name += ".primary"
			case TypeDeposed:
				name += ".deposed"
			case TypeTainted:
				name += ".tainted"
			}
		}

		if r.Index >= 0 {
			name += fmt.Sprintf("[%d]", r.Index)
		}
		result = append(result, name)
	}

	return strings.Join(result, ".")
}

func ParseResourceAddress(s string) (*ResourceAddress, error) {
	matches, err := tokenizeResourceAddress(s)
	if err != nil {
		return nil, err
	}
	mode := config.ManagedResourceMode
	if matches["data_prefix"] != "" {
		mode = config.DataResourceMode
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

	// not allowed to say "data." without a type following
	if mode == config.DataResourceMode && matches["type"] == "" {
		return nil, fmt.Errorf("must target specific data instance")
	}

	return &ResourceAddress{
		Path:            path,
		Index:           resourceIndex,
		InstanceType:    instanceType,
		InstanceTypeSet: matches["instance_type"] != "",
		Name:            matches["name"],
		Type:            matches["type"],
		Mode:            mode,
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

	// mode is significant only when type is set
	modeMatch := addr.Type == "" ||
		other.Type == "" ||
		addr.Mode == other.Mode

	return pathMatch &&
		indexMatch &&
		addr.InstanceType == other.InstanceType &&
		nameMatch &&
		typeMatch &&
		modeMatch
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
		// possibly "data.", if targeting is a data resource
		`(?P<data_prefix>(?:data\.)?)` +
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
