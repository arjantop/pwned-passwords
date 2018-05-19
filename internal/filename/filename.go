package filename

import "path"

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

func PathFor(s string, extension string) string {
	parts := splitEqualLength(s, splitPartLength)
	return path.Join(parts...) + extension
}
