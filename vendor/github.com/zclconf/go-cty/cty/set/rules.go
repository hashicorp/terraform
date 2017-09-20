package set

// Rules represents the operations that define membership for a Set.
//
// Each Set has a Rules instance, whose methods must satisfy the interface
// contracts given below for any value that will be added to the set.
type Rules interface {
	// Hash returns an int that somewhat-uniquely identifies the given value.
	//
	// A good hash function will minimize collisions for values that will be
	// added to the set, though collisions *are* permitted. Collisions will
	// simply reduce the efficiency of operations on the set.
	Hash(interface{}) int

	// Equivalent returns true if and only if the two values are considered
	// equivalent for the sake of set membership. Two values that are
	// equivalent cannot exist in the set at the same time, and if two
	// equivalent values are added it is undefined which one will be
	// returned when enumerating all of the set members.
	//
	// Two values that are equivalent *must* result in the same hash value,
	// though it is *not* required that two values with the same hash value
	// be equivalent.
	Equivalent(interface{}, interface{}) bool
}
