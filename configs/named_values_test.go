package configs

import (
	"testing"
)

func Test_looksLikeSentences(t *testing.T) {
	tests := map[string]struct {
		args string
		want bool
	}{
		"empty sentence": {
			args: "",
			want: false,
		},
		"valid sentence": {
			args: "A valid sentence.",
			want: true,
		},
		"valid sentence with an accent": {
			args: `A Valid sentence with an accent "é".`,
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := looksLikeSentences(tt.args); got != tt.want {
				t.Errorf("looksLikeSentences() = %v, want %v", got, tt.want)
			}
		})
	}
}
