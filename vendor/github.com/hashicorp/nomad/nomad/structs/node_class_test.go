package structs

import (
	"reflect"
	"testing"
)

func testNode() *Node {
	return &Node{
		ID:         GenerateUUID(),
		Datacenter: "dc1",
		Name:       "foobar",
		Attributes: map[string]string{
			"kernel.name": "linux",
			"arch":        "x86",
			"version":     "0.1.0",
			"driver.exec": "1",
		},
		Resources: &Resources{
			CPU:      4000,
			MemoryMB: 8192,
			DiskMB:   100 * 1024,
			IOPS:     150,
			Networks: []*NetworkResource{
				&NetworkResource{
					Device: "eth0",
					CIDR:   "192.168.0.100/32",
					MBits:  1000,
				},
			},
		},
		Links: map[string]string{
			"consul": "foobar.dc1",
		},
		Meta: map[string]string{
			"pci-dss": "true",
		},
		NodeClass: "linux-medium-pci",
		Status:    NodeStatusReady,
	}
}

func TestNode_ComputedClass(t *testing.T) {
	// Create a node and gets it computed class
	n := testNode()
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	old := n.ComputedClass

	// Compute again to ensure determinism
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if old != n.ComputedClass {
		t.Fatalf("ComputeClass() should have returned same class; got %v; want %v", n.ComputedClass, old)
	}

	// Modify a field and compute the class again.
	n.Datacenter = "New DC"
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}

	if old == n.ComputedClass {
		t.Fatal("ComputeClass() returned same computed class")
	}
}

func TestNode_ComputedClass_Ignore(t *testing.T) {
	// Create a node and gets it computed class
	n := testNode()
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	old := n.ComputedClass

	// Modify an ignored field and compute the class again.
	n.ID = "New ID"
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}

	if old != n.ComputedClass {
		t.Fatal("ComputeClass() should have ignored field")
	}
}

func TestNode_ComputedClass_Attr(t *testing.T) {
	// Create a node and gets it computed class
	n := testNode()
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	old := n.ComputedClass

	// Add a unique addr and compute the class again
	n.Attributes["unique.foo"] = "bar"
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if old != n.ComputedClass {
		t.Fatal("ComputeClass() didn't ignore unique attr suffix")
	}

	// Modify an attribute and compute the class again.
	n.Attributes["version"] = "New Version"
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	if old == n.ComputedClass {
		t.Fatal("ComputeClass() ignored attribute change")
	}

	// Remove and attribute and compute the class again.
	old = n.ComputedClass
	delete(n.Attributes, "driver.exec")
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputedClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	if old == n.ComputedClass {
		t.Fatalf("ComputedClass() ignored removal of attribute key")
	}
}

func TestNode_ComputedClass_Meta(t *testing.T) {
	// Create a node and gets it computed class
	n := testNode()
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	old := n.ComputedClass

	// Modify a meta key and compute the class again.
	n.Meta["pci-dss"] = "false"
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	if old == n.ComputedClass {
		t.Fatal("ComputeClass() ignored meta change")
	}
	old = n.ComputedClass

	// Add a unique meta key and compute the class again.
	n.Meta["unique.foo"] = "ignore"
	if err := n.ComputeClass(); err != nil {
		t.Fatalf("ComputeClass() failed: %v", err)
	}
	if n.ComputedClass == "" {
		t.Fatal("ComputeClass() didn't set computed class")
	}
	if old != n.ComputedClass {
		t.Fatal("ComputeClass() didn't ignore unique meta key")
	}
}

func TestNode_EscapedConstraints(t *testing.T) {
	// Non-escaped constraints
	ne1 := &Constraint{
		LTarget: "${attr.kernel.name}",
		RTarget: "linux",
		Operand: "=",
	}
	ne2 := &Constraint{
		LTarget: "${meta.key_foo}",
		RTarget: "linux",
		Operand: "<",
	}
	ne3 := &Constraint{
		LTarget: "${node.dc}",
		RTarget: "test",
		Operand: "!=",
	}

	// Escaped constraints
	e1 := &Constraint{
		LTarget: "${attr.unique.kernel.name}",
		RTarget: "linux",
		Operand: "=",
	}
	e2 := &Constraint{
		LTarget: "${meta.unique.key_foo}",
		RTarget: "linux",
		Operand: "<",
	}
	e3 := &Constraint{
		LTarget: "${unique.node.id}",
		RTarget: "test",
		Operand: "!=",
	}
	constraints := []*Constraint{ne1, ne2, ne3, e1, e2, e3}
	expected := []*Constraint{ne1, ne2, ne3}
	if act := EscapedConstraints(constraints); reflect.DeepEqual(act, expected) {
		t.Fatalf("EscapedConstraints(%v) returned %v; want %v", constraints, act, expected)
	}
}
