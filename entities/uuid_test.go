package entities

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateUUID(t *testing.T) {
	id := uuid.New()

	pid, _, err := ValidateUUID(id.String())

	if id != pid {
		t.Errorf("id and pid are not equals")
	}

	if err != nil {
		t.Error(err)
	}
}
