package util

import (
	"log"
	"os"
)

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

// FileReader takes a string filename and retuns a byte slice and error
func FileReader(filename string) ([]byte, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("could not read file: %s.\nError:%v\n", filename, err)
	}
	return file, err
}
