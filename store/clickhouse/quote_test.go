package clickhouse

import "testing"

// ClickHouse reads backslash escapes inside backtick-quoted identifiers, so a
// name that ends a segment in a backslash must not be able to swallow the
// doubled backtick that follows it.
func TestQuoteIdentEscapesBackslashes(t *testing.T) {
	for in, want := range map[string]string{
		"plain":                     "`plain`",
		"a`b":                       "`a``b`",
		`a\b`:                       "`a\\\\b`",
		"x\\`,(SELECT 42) AS y -- ": "`x\\\\``,(SELECT 42) AS y -- `",
	} {
		if got := quoteIdent(in); got != want {
			t.Errorf("quoteIdent(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestColumnQuotesEverySegment(t *testing.T) {
	got, err := column("log.a\\`b")
	if err != nil {
		t.Fatal(err)
	}
	if want := "`log`.`a\\\\``b`"; got != want {
		t.Errorf("column = %s, want %s", got, want)
	}
}

func TestQuoteStringEscapesBackslashes(t *testing.T) {
	if got, want := quoteString(`a\'b`), `'a\\\'b'`; got != want {
		t.Errorf("quoteString = %s, want %s", got, want)
	}
}
