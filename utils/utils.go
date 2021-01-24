package utils

import (
	"strings"
	"unicode"
)

// FormatHex makes hex values more readable
// For debugging purposes, it is an inefficient implementation
func FormatHex(instr string) (outstr string) {
	outstr = ""
	for i := range instr {
		if i%2 == 0 {
			outstr += instr[i:i+2] + " "
		}
	}
	return
}

// CleanString cleans up the non-ASCII characters from an input string
func CleanString(input string) string {
	return strings.TrimFunc(input, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
}
