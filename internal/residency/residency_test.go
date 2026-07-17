package residency

import "testing"

func TestSelfHosted(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"self_hosted", true},
		{"SELF_HOSTED", true},
		{" self_hosted ", true},
		{"", false},
		{"cloud", false},
		{"selfhosted", false},
	}
	for _, tc := range cases {
		t.Setenv(EnvVar, tc.value)
		if got := SelfHosted(); got != tc.want {
			t.Errorf("SelfHosted() with %q = %v, want %v", tc.value, got, tc.want)
		}
	}
}
