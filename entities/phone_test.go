package entities

import "testing"

func TestValidatePhone(t *testing.T) {
	var validPhones = []string{
		"+11-2222-3333",
		"+1 (222) 333-4444",
		"+111 2222 3333",
		"+1 (123) 12-2-333",
	}

	var invalidPhones = []string{
		"+ 11-2222-3333",
		" +11-2222-3333",
		"+11-2222-3333 ",
		"+1 1 222 2-3333",
		"+1 (12 3) 12-2-333",
	}

	for _, p := range validPhones {
		vp, _, err := ValidatePhone(p)
		if err != nil {
			t.Error(err)
		}

		if vp != p {
			t.Errorf("phones are not equals")
		}
	}

	for _, p := range invalidPhones {
		vp, _, err := ValidatePhone(p)
		if err == nil {
			t.Error("this should return an error")
		}

		if vp != "" {
			t.Error("phone should be empty")
		}
	}
}
