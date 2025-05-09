package entities

// ValidatePhone validates a phone number and returns the validated phone number and its SHA3-256 hash.
// If the value is not a string, it returns an error.
func ValidatePhone(value string) (string, string, error) {
	e := ValidateRegEx(`^([+][1-9]{1,1}[0-9]{0,2})([\s]?[(][1-9]{1,1}[0-9]{0,3}[)])?([\s]?[-]?[0-9]{1,4}){1,3}$`, value)
	if e != nil {
		return "", "", e
	}

	return value, GenerateSHA3256(value), nil
}
