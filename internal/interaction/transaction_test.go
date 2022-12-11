package interaction

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomDigits(t *testing.T) {
	tests := []struct {
		name              string
		inputCount        int
		expectedStringLen int
	}{
		{
			name:              "Should return string with len 4",
			inputCount:        4,
			expectedStringLen: 4,
		},
		{
			name:              "Should return empty string when len is negative",
			inputCount:        -1,
			expectedStringLen: 0,
		},
		{
			name:              "Should return empty string when len is zero",
			inputCount:        0,
			expectedStringLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := randomDigits(tt.inputCount)

			require.Len(t, res, tt.expectedStringLen)
			if tt.expectedStringLen > 0 {
				require.Regexp(t, regexp.MustCompile("[0-9]+"), res)
			}
		})
	}
}
