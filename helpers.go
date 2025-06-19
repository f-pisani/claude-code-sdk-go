package claudecode

// SafeBoolPtr safely dereferences a bool pointer, returning false if nil
func SafeBoolPtr(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// SafeFloat64Ptr safely dereferences a float64 pointer, returning 0 if nil
func SafeFloat64Ptr(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// SafeIntPtr safely dereferences an int pointer, returning 0 if nil
func SafeIntPtr(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// SafeStringPtr safely dereferences a string pointer, returning empty string if nil
func SafeStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// BoolPtr returns a pointer to the given bool value
func BoolPtr(b bool) *bool {
	return &b
}

// Float64Ptr returns a pointer to the given float64 value
func Float64Ptr(f float64) *float64 {
	return &f
}

// IntPtr returns a pointer to the given int value
func IntPtr(i int) *int {
	return &i
}

// StringPtr returns a pointer to the given string value
func StringPtr(s string) *string {
	return &s
}
