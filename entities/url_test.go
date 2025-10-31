package entities

import "testing"

func TestValidateURL(t *testing.T) {
	sample := "https://www.example.com/heLLo"

	purl, _, err := ValidateURL(sample)

	if purl != sample {
		t.Errorf("purl and sample are not equals")
	}

	if err != nil {
		t.Error(err)
	}
}
