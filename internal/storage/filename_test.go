package storage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathFor(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedResult string
	}{
		{"shorter than part length", "ab", "ab.bin"},
		{"exactly part length", "abc", "abc.bin"},
		{"longer than part length", "abcde", "abc/de.bin"},
		{"multiple of part length", "abcdef", "abc/def.bin"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedResult, PathFor(tc.input, ".bin"))
		})
	}
}

func TestPathForWithEmptyExtension(t *testing.T) {
	assert.Equal(t, "aa", PathFor("aa", ""))
}

func ExamplePathFor() {
	fmt.Println(PathFor("abcde", ".txt"))
	// Output: abc/de.txt
}
