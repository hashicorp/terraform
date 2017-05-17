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
	if r == nil {
		return nil
	}

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

// stateId returns the ID that this resource should be entered with
// in the state. This is also used for diffs. In the future, we'd like to
// move away from this string field so I don't export this.
func (r *ResourceAddress) stateId() string {
	result := fmt.Sprintf("%s.%s", r.Type, r.Name)
	switch r.Mode {
	case config.ManagedResourceMode:
		// Done
	case config.DataResourceMode:
		result = fmt.Sprintf("data.%s", result)
	default:
		panic(fmt.Errorf("unknown resource mode: %s", r.Mode))
	}
	if r.Index >= 0 {
		result += fmt.Sprintf(".%d", r.Index)
	}

	return result
}

// parseResourceAddressConfig creates a resource address from a config.Resource
func parseResourceAddressConfig(r *config.Resource) (*ResourceAddress, error) {
	return &ResourceAddress{
		Type:         r.Type,
		Name:         r.Name,
		Index:        -1,
		InstanceType: TypePrimary,
		Mode:         r.Mode,
	}, nil
}

// parseResourceAddressInternal parses the somewhat bespoke resource
// identifier used in states and diffs, such as "instance.name.0".
func parseResourceAddressInternal(s string) (*ResourceAddress, error) {
	// Split based on ".". Every resource address should have at least two
	// elements (type and name).
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return nil, fmt.Errorf("Invalid internal resource address format: %s", s)
	}

	// Data resource if we have at least 3 parts and the first one is data
	mode := config.ManagedResourceMode
	if len(parts) > 2 && parts[0] == "data" {
		mode = config.DataResourceMode
		parts = parts[1:]
	}

	// If we're not a data resource and we have more than 3, then it is an error
	if len(parts) > 3 && mode != config.DataResourceMode {
		return nil, fmt.Errorf("Invalid internal resource address format: %s", s)
	}

	// Build the parts of the resource address that are guaranteed to exist
	addr := &ResourceAddress{
		Type:         parts[0],
		Name:         parts[1],
		Index:        -1,
		InstanceType: TypePrimary,
		Mode:         mode,
	}

	// If we have more parts, then we have an index. Parse that.
	if len(parts) > 2 {
		idx, err := strconv.ParseInt(parts[2], 0, 0)
		if err != nil {
			return nil, fmt.Errorf("Error parsing resource address %q: %s", s, err)
		}

		addr.Index = int(idx)
	}

	return addr, nil
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
