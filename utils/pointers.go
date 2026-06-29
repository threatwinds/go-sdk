package utils

// PointerOf takes a value of any type and returns a pointer to that value.
// This is useful for creating pointers to literals or values that are not
// already pointers.
//
// Example usage:
//
//	intValue := 42
//	intPointer := PointerOf(intValue)
//
// Type Parameters:
//
//	t: The type of the value to be pointed to.
//
// Parameters:
//
//	s: The value to create a pointer for.
//
// Returns:
//
//	A pointer to the provided value.
func PointerOf[t any](s t) *t {
	return &s
}
