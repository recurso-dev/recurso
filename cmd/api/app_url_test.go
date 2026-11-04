package main

import "testing"

func TestAppBaseURL(t *testing.T) {
	const api = "https://api.test"
	for _, tc := range []struct {
		name, dashboard, portal, want string
	}{
		{"nothing set falls back to the API for local dev", "", "", api},
		{"dashboard", "https://app.test", "", "https://app.test"},
		{"portal alone", "", "https://portal.test", "https://portal.test"},
		{"dashboard wins", "https://app.test", "https://portal.test", "https://app.test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DASHBOARD_URL", tc.dashboard)
			t.Setenv("PORTAL_URL", tc.portal)
			if got := appBaseURL(api); got != tc.want {
				t.Errorf("appBaseURL = %q, want %q", got, tc.want)
			}
		})
	}
}
