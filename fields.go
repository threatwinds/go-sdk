package go_sdk

import "errors"

// ValidateReservedField validates a field to ensure it is not empty or a reserved field.
func ValidateReservedField(f string) error {
	switch f {
	case "":
		return errors.New("field cannot be empty")
	case "raw":
		return errors.New("field cannot be 'raw'")
	case "dataType":
		return errors.New("field cannot be 'dataType'")
	case "@timestamp":
		return errors.New("field cannot be '@timestamp'")
	case "dataSource":
		return errors.New("field cannot be 'dataSource'")
	}

	return nil
}