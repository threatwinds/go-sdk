package entities

import "testing"

func TestValidateRegexComp(t *testing.T) {
	v := `^(?i)[a-z0-9]+(([-]{1,2}[a-z0-9]+)*([\.]{1}[a-z0-9]+)*)*(\.[a-z]{2,20})$`
	i := `^(?i)[a-z0-9]+(([-]{1,2}[a-z0-9]+)*([\.]{1}[a-z0-9]+)**(\.[a-z]{2,20})$`
	r, _, err := ValidateRegexComp(v)
	if err != nil {
		t.Error(err.Error())
	}
	if r != v {
		t.Errorf("v and r are not equals")
	}
	_, _, err = ValidateRegexComp(i)
	if err == nil {
		t.Errorf("must fail")
	}
}
