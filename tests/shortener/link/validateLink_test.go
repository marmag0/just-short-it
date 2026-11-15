package link_test

import (
	"strings"
	"testing"

	"github.com/marmag0/just-short-it/services/shortener/link"
)

func TestInitialLinkValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"min length", "abc", true},
		{"max length", strings.Repeat("a", 32), true},
		{"mixed case and digits", "Ab3dE9", true},
		{"digits only", "123", true},
		{"empty", "", false},
		{"too short", "ab", false},
		{"too long", strings.Repeat("a", 33), false},
		{"dash", "ab-cd", false},
		{"underscore", "ab_cd", false},
		{"dot", "ab.cd", false},
		{"space", "ab cd", false},
		{"slash", "ab/cd", false},
		{"percent encoding", "ab%20cd", false},
		{"non-ASCII letters", "żółw", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, msg := link.InitialLinkValid(tt.input)
			if ok != tt.want {
				t.Errorf("InitialLinkValid(%q) = %v, want %v", tt.input, ok, tt.want)
			}
			if ok != (msg == "") {
				t.Errorf("InitialLinkValid(%q) message = %q, want a message only for invalid links", tt.input, msg)
			}
		})
	}
}
