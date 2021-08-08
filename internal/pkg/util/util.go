package util

// StrToPtr takes a string and returns a reference
func StrToPtr(s string) *string {
	return &s
}

// IntToPtr takes an int and returns a reference
func IntToPtr(i int) *int {
	return &i
}

// Uint32ToPtr takes an uint32 and return a reference
func Uint32ToPtr(i uint32) *uint32 {
	return &i
}
