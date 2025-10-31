package entities

// ValidateInteger validates if a value is an integer and returns its int64 representation,
// its SHA3-256 hash and an error if the value is not an integer.
func ValidateInteger(value int64) (int64, string, error) {
	return value, GenerateSHA3256(value), nil
}
