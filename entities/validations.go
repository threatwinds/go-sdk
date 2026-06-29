package entities

import (
	"fmt"
	"github.com/threatwinds/go-sdk/utils"
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
				return ValidateString(utils.CastString(value), false)
			case ISTR:
				return ValidateString(utils.CastString(value), true)
			case IP:
				return ValidateIP(utils.CastString(value))
			case EMAIL:
				return ValidateEmail(utils.CastString(value))
			case FQDN:
				return ValidateFQDN(utils.CastString(value))
			case INTEGER:
				return ValidateInteger(utils.CastInt64(value))
			case CIDR:
				return ValidateCIDR(utils.CastString(value))
			case CITY:
				return ValidateCity(utils.CastString(value))
			case COUNTRY:
				return ValidateCountry(utils.CastString(value))
			case FLOAT:
				return ValidateFloat(utils.CastFloat64(value))
			case BOOLEAN:
				return ValidateBoolean(utils.CastBool(value))
			case URL:
				return ValidateURL(utils.CastString(value))
			case MD5:
				return ValidateMD5(utils.CastString(value))
			case HEXADECIMAL:
				return ValidateHexadecimal(utils.CastString(value))
			case BASE64:
				return ValidateBase64(utils.CastString(value))
			case DATE:
				return ValidateDate(utils.CastString(value))
			case MAC:
				return ValidateMAC(utils.CastString(value))
			case MIME:
				return ValidateMime(utils.CastString(value))
			case PHONE:
				return ValidatePhone(utils.CastString(value))
			case SHA1:
				return ValidateSHA1(utils.CastString(value))
			case SHA224:
				return ValidateSHA224(utils.CastString(value))
			case SHA256:
				return ValidateSHA256(utils.CastString(value))
			case SHA384:
				return ValidateSHA384(utils.CastString(value))
			case SHA512:
				return ValidateSHA512(utils.CastString(value))
			case SHA3_224:
				return ValidateSHA3224(utils.CastString(value))
			case SHA3_256:
				return ValidateSHA3256(utils.CastString(value))
			case SHA3_384:
				return ValidateSHA3384(utils.CastString(value))
			case SHA3_512:
				return ValidateSHA3512(utils.CastString(value))
			case SHA512_224:
				return ValidateSHA512224(utils.CastString(value))
			case SHA512_256:
				return ValidateSHA512256(utils.CastString(value))
			case DATETIME:
				return ValidateDatetime(utils.CastString(value))
			case UUID:
				return ValidateUUID(utils.CastString(value))
			case PATH:
				return ValidatePath(utils.CastString(value))
			case IDENTIFIER:
				return ValidateIdentifier(utils.CastString(value))
			case ADVERSARY:
				return ValidateAdversary(utils.CastString(value))
			case REGEX:
				return ValidateRegexComp(utils.CastString(value))
			case PORT:
				return ValidatePort(utils.CastString(value))
			default:
				return nil, "", fmt.Errorf("unknown validator for value: %v", value)
			}
		}
	}
	return nil, "", fmt.Errorf("unknown type: %s", t)
}
