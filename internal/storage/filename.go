package storage

import (
	"path"
)

// splitEqualLength splits a string into strings of equal length.
//
// The last string part will always have a length greater than zero and
// less or equal than partLen.
func splitEqualLength(s string, partLen int) []string {
	var result []string

	for len(s) > 0 {
		availableLength := partLen
		if len(s) < partLen {
			availableLength = len(s)
		}

		result = append(result, s[:availableLength])
		s = s[availableLength:]
	}

	return result
}

const splitPartLength = 3

// PathFor takes a string and an extension and generates a file path.
func PathFor(s string, extension string) string {
	parts := splitEqualLength(s, splitPartLength)
	return path.Join(parts...) + extension
}
