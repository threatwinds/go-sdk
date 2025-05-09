package entities

import (
	"fmt"
)

// ValidateValue validates a value against a specified type.
// It takes a value of any type and a string representing the type to validate against.
// It returns the validated value, its SHA3-256 hash, and an error if validation fails.
// The function looks up the type in the Definitions and calls the appropriate validation function.
func ValidateValue(value interface{}, t string) (interface{}, string, error) {
	for _, def := range Definitions {
		if def.Type == t {
			switch def.DataType {
			case STR:
				return ValidateString(value, false)
			case ISTR:
				return ValidateString(value, true)
			case IP:
				return ValidateIP(value)
			case EMAIL:
				return ValidateEmail(value)
			case FQDN:
				return ValidateFQDN(value)
			case INTEGER:
				return ValidateInteger(value)
			case CIDR:
				return ValidateCIDR(value)
			case CITY:
				return ValidateCity(value)
			case COUNTRY:
				return ValidateCountry(value)
			case FLOAT:
				return ValidateFloat(value)
			case BOOLEAN:
				return ValidateBoolean(value)
			case URL:
				return ValidateURL(value)
			case MD5:
				return ValidateMD5(value)
			case HEXADECIMAL:
				return ValidateHexadecimal(value)
			case BASE64:
				return ValidateBase64(value)
			case DATE:
				return ValidateDate(value)
			case MAC:
				return ValidateMAC(value)
			case MIME:
				return ValidateMime(value)
			case PHONE:
				return ValidatePhone(value)
			case SHA1:
				return ValidateSHA1(value)
			case SHA224:
				return ValidateSHA224(value)
			case SHA256:
				return ValidateSHA256(value)
			case SHA384:
				return ValidateSHA384(value)
			case SHA512:
				return ValidateSHA512(value)
			case SHA3_224:
				return ValidateSHA3224(value)
			case SHA3_256:
				return ValidateSHA3256(value)
			case SHA3_384:
				return ValidateSHA3384(value)
			case SHA3_512:
				return ValidateSHA3512(value)
			case SHA512_224:
				return ValidateSHA512224(value)
			case SHA512_256:
				return ValidateSHA512256(value)
			case DATETIME:
				return ValidateDatetime(value)
			case UUID:
				return ValidateUUID(value)
			case PATH:
				return ValidatePath(value)
			case IDENTIFIER:
				return ValidateIdentifier(value)
			case ADVERSARY:
				return ValidateAdversary(value)
			case REGEX:
				return ValidateRegexComp(value)
			case PORT:
				return ValidatePort(value)
			default:
				return nil, "", fmt.Errorf("unknown validator for value: %v", value)
			}
		}
	}
	return nil, "", fmt.Errorf("unknown type: %s", t)
}
