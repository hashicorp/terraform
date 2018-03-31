package addrs

// Self is the address of the special object "self" that behaves as an alias
// for a containing object currently in scope.
const Self selfT = 0

type selfT int

func (s selfT) referenceableSigil() {
}
