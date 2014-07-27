package depgraph

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

// ParseNouns is used to parse a string in the format of:
// a -> b ; edge name
// b -> c
// Into a series of nouns and dependencies
func ParseNouns(s string) map[string]*Noun {
	lines := strings.Split(s, "\n")
	nodes := make(map[string]*Noun)
	for _, line := range lines {
		var edgeName string
		if idx := strings.Index(line, ";"); idx >= 0 {
			edgeName = strings.Trim(line[idx+1:], " \t\r\n")
			line = line[:idx]
		}
		parts := strings.SplitN(line, "->", 2)
		if len(parts) != 2 {
			continue
		}
		head_name := strings.Trim(parts[0], " \t\r\n")
		tail_name := strings.Trim(parts[1], " \t\r\n")
		head := nodes[head_name]
		if head == nil {
			head = &Noun{Name: head_name}
			nodes[head_name] = head
		}
		tail := nodes[tail_name]
		if tail == nil {
			tail = &Noun{Name: tail_name}
			nodes[tail_name] = tail
		}
		edge := &Dependency{
			Name:   edgeName,
			Source: head,
			Target: tail,
		}
		head.Deps = append(head.Deps, edge)
	}
	return nodes
}

func NounMapToList(m map[string]*Noun) []*Noun {
	list := make([]*Noun, 0, len(m))
	for _, n := range m {
		list = append(list, n)
	}
	return list
}

func TestGraph_Noun(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
b -> e
c -> d
c -> e`)

	g := &Graph{
		Name:  "Test",
		Nouns: NounMapToList(nodes),
	}

	n := g.Noun("a")
	if n == nil {
		t.Fatal("should not be nil")
	}
	if n.Name != "a" {
		t.Fatalf("bad: %#v", n)
	}
}

func TestGraph_String(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
b -> e
c -> d
c -> e`)

	g := &Graph{
		Name:  "Test",
		Nouns: NounMapToList(nodes),
		Root:  nodes["a"],
	}
	actual := g.String()

	expected := `
root: a
a
  a -> b
  a -> c
b
  b -> d
  b -> e
c
  c -> d
  c -> e
d
e
`

	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	if actual != expected {
		t.Fatalf("bad:\n%s\n!=\n%s", actual, expected)
	}
}

func TestGraph_Validate(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
b -> e
c -> d
c -> e`)
	list := NounMapToList(nodes)

	g := &Graph{Name: "Test", Nouns: list}
	if err := g.Validate(); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestGraph_Validate_Cycle(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
d -> b`)
	list := NounMapToList(nodes)

	g := &Graph{Name: "Test", Nouns: list}
	err := g.Validate()
	if err == nil {
		t.Fatalf("expected err")
	}

	vErr, ok := err.(*ValidateError)
	if !ok {
		t.Fatalf("expected validate error")
	}

	if len(vErr.Cycles) != 1 {
		t.Fatalf("expected cycles")
	}

	cycle := vErr.Cycles[0]
	cycleNodes := make([]string, len(cycle))
	for i, c := range cycle {
		cycleNodes[i] = c.Name
	}
	sort.Strings(cycleNodes)

	if cycleNodes[0] != "b" {
		t.Fatalf("bad: %v", cycle)
	}
	if cycleNodes[1] != "d" {
		t.Fatalf("bad: %v", cycle)
	}
}

func TestGraph_Validate_MultiRoot(t *testing.T) {
	nodes := ParseNouns(`a -> b
c -> d`)
	list := NounMapToList(nodes)

	g := &Graph{Name: "Test", Nouns: list}
	err := g.Validate()
	if err == nil {
		t.Fatalf("expected err")
	}

	vErr, ok := err.(*ValidateError)
	if !ok {
		t.Fatalf("expected validate error")
	}

	if !vErr.MissingRoot {
		t.Fatalf("expected missing root")
	}
}

func TestGraph_Validate_NoRoot(t *testing.T) {
	nodes := ParseNouns(`a -> b
b -> a`)
	list := NounMapToList(nodes)

	g := &Graph{Name: "Test", Nouns: list}
	err := g.Validate()
	if err == nil {
		t.Fatalf("expected err")
	}

	vErr, ok := err.(*ValidateError)
	if !ok {
		t.Fatalf("expected validate error")
	}

	if !vErr.MissingRoot {
		t.Fatalf("expected missing root")
	}
}

func TestGraph_Validate_Unreachable(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
x -> x`)
	list := NounMapToList(nodes)

	g := &Graph{Name: "Test", Nouns: list}
	err := g.Validate()
	if err == nil {
		t.Fatalf("expected err")
	}

	vErr, ok := err.(*ValidateError)
	if !ok {
		t.Fatalf("expected validate error")
	}

	if len(vErr.Unreachable) != 1 {
		t.Fatalf("expected unreachable")
	}

	if vErr.Unreachable[0].Name != "x" {
		t.Fatalf("bad: %v", vErr.Unreachable[0])
	}
}

type VersionMeta int
type VersionConstraint struct {
	Min int
	Max int
}

func (v *VersionConstraint) Satisfied(head, tail *Noun) (bool, error) {
	vers := int(tail.Meta.(VersionMeta))
	if vers < v.Min {
		return false, fmt.Errorf("version %d below minimum %d",
			vers, v.Min)
	} else if vers > v.Max {
		return false, fmt.Errorf("version %d above maximum %d",
			vers, v.Max)
	}
	return true, nil
}

func (v *VersionConstraint) String() string {
	return "version"
}

func TestGraph_ConstraintViolation(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
b -> e
c -> d
c -> e`)
	list := NounMapToList(nodes)

	// Add a version constraint
	vers := &VersionConstraint{1, 3}

	// Introduce some constraints
	depB := nodes["a"].Deps[0]
	depB.Constraints = []Constraint{vers}
	depC := nodes["a"].Deps[1]
	depC.Constraints = []Constraint{vers}

	// Add some versions
	nodes["b"].Meta = VersionMeta(0)
	nodes["c"].Meta = VersionMeta(4)

	g := &Graph{Name: "Test", Nouns: list}
	err := g.Validate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = g.CheckConstraints()
	if err == nil {
		t.Fatalf("Expected err")
	}

	cErr, ok := err.(*ConstraintError)
	if !ok {
		t.Fatalf("expected constraint error")
	}

	if len(cErr.Violations) != 2 {
		t.Fatalf("expected 2 violations: %v", cErr)
	}

	if cErr.Violations[0].Error() != "Constraint version between a and b violated: version 0 below minimum 1" {
		t.Fatalf("err: %v", cErr.Violations[0])
	}

	if cErr.Violations[1].Error() != "Constraint version between a and c violated: version 4 above maximum 3" {
		t.Fatalf("err: %v", cErr.Violations[1])
	}
}

func TestGraph_Constraint_NoViolation(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
b -> e
c -> d
c -> e`)
	list := NounMapToList(nodes)

	// Add a version constraint
	vers := &VersionConstraint{1, 3}

	// Introduce some constraints
	depB := nodes["a"].Deps[0]
	depB.Constraints = []Constraint{vers}
	depC := nodes["a"].Deps[1]
	depC.Constraints = []Constraint{vers}

	// Add some versions
	nodes["b"].Meta = VersionMeta(2)
	nodes["c"].Meta = VersionMeta(3)

	g := &Graph{Name: "Test", Nouns: list}
	err := g.Validate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = g.CheckConstraints()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestGraphWalk(t *testing.T) {
	nodes := ParseNouns(`a -> b
a -> c
b -> d
b -> e
c -> d
c -> e`)
	list := NounMapToList(nodes)
	g := &Graph{Name: "Test", Nouns: list}
	if err := g.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}

	var namesLock sync.Mutex
	names := make([]string, 0, 0)
	err := g.Walk(func(n *Noun) error {
		namesLock.Lock()
		defer namesLock.Unlock()
		names = append(names, n.Name)
		return nil
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := [][]string{
		{"e", "d", "c", "b", "a"},
		{"e", "d", "b", "c", "a"},
		{"d", "e", "c", "b", "a"},
		{"d", "e", "b", "c", "a"},
	}
	found := false
	for _, expect := range expected {
		if reflect.DeepEqual(expect, names) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("bad: %#v", names)
	}
}

func TestGraphWalk_error(t *testing.T) {
	nodes := ParseNouns(`a -> b
b -> c
a -> d`)
	list := NounMapToList(nodes)
	g := &Graph{Name: "Test", Nouns: list}
	if err := g.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// We repeat this a lot because sometimes timing causes
	// a false positive.
	for i := 0; i < 100; i++ {
		var lock sync.Mutex
		var walked []string
		err := g.Walk(func(n *Noun) error {
			lock.Lock()
			defer lock.Unlock()

			walked = append(walked, n.Name)

			if n.Name == "b" {
				return fmt.Errorf("foo")
			}

			return nil
		})
		if err == nil {
			t.Fatal("should error")
		}

		sort.Strings(walked)

		expected := []string{"b", "c", "d"}
		if !reflect.DeepEqual(walked, expected) {
			t.Fatalf("bad: %#v", walked)
		}
	}
}
