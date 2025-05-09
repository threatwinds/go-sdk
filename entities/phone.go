package entities

import "fmt"

// ValidatePhone validates a phone number and returns the validated phone number and its SHA3-256 hash.
// If the value is not a string, it returns an error.
func ValidatePhone(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	e := ValidateRegEx(`^([+][1-9]{1,1}[0-9]{0,2})([\s]?[(][1-9]{1,1}[0-9]{0,3}[)])?([\s]?[-]?[0-9]{1,4}){1,3}$`, v)
	if e != nil {
		return "", "", e
	}

	return v, GenerateSHA3256(v), nil
}
