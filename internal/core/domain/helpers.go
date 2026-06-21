package domain

// StringPtr returns a pointer to the passed string
func StringPtr(s string) *string {
	return &s
}

// PtrToString returns the string value or empty string if nil
func PtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PtrStringPtr returns nil if string is empty, else returns pointer to string
func PtrStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
