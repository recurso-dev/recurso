package httpsafe

import "testing"

// TestValidateExternalURL covers the SSRF guard: internal/private/link-local
// hosts and non-http schemes are rejected; public IP hosts pass. IP literals are
// used so the test does no DNS.
func TestValidateExternalURL(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/x",                 // loopback
		"http://localhost/x",                 // loopback name (resolves to 127.0.0.1/::1)
		"http://169.254.169.254/latest/meta", // cloud metadata (link-local)
		"http://10.0.0.5/admin",              // private
		"http://192.168.1.1/",                // private
		"http://172.16.0.1/",                 // private
		"http://[::1]/x",                     // ipv6 loopback
		"http://0.0.0.0/x",                   // unspecified
		"ftp://8.8.8.8/x",                    // non-http scheme
		"file:///etc/passwd",                 // non-http scheme
		"gopher://8.8.8.8/x",                 // non-http scheme
		"not a url",                          // unparseable
	}
	for _, u := range blocked {
		if err := ValidateExternalURL(u); err == nil {
			t.Errorf("ValidateExternalURL(%q) = nil, want rejected", u)
		}
	}

	allowed := []string{
		"http://8.8.8.8/webhook",
		"https://93.184.216.34/hook", // a public IP literal
		"https://1.1.1.1/",
	}
	for _, u := range allowed {
		if err := ValidateExternalURL(u); err != nil {
			t.Errorf("ValidateExternalURL(%q) = %v, want allowed", u, err)
		}
	}
}
