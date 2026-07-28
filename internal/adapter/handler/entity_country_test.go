package handler

import "testing"

// The entity endpoint historically took country_code while /auth/register
// takes country; a caller who used one and then the other silently got an
// empty country. toInput accepts either, with country_code winning.
func TestEntityRequestCountryAlias(t *testing.T) {
	cases := []struct {
		name string
		req  entityRequest
		want string
	}{
		{"country_code only", entityRequest{CountryCode: "DE"}, "DE"},
		{"country alias only", entityRequest{Country: "US"}, "US"},
		{"country_code wins over country", entityRequest{CountryCode: "DE", Country: "US"}, "DE"},
		{"neither", entityRequest{}, ""},
	}
	for _, c := range cases {
		if got := c.req.toInput().CountryCode; got != c.want {
			t.Errorf("%s: CountryCode = %q, want %q", c.name, got, c.want)
		}
	}
}
