package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "https://example.com", want: true},
		{input: "http://localhost:8080", want: true},
		{input: "ftp://ftp.example.com"}, // not http(s)
		{input: "file:///tmp/file.txt"},  // not http(s)
		{input: "//example.com"},         // missing scheme
		{input: "example.com"},           // missing scheme
		{input: "/some/path"},            // path only
		{input: ""},                      // empty string
		{input: "not a url"},             // invalid
		{input: "https:/example.com"},    // malformed scheme
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsURL(tt.input)
			assert.Equal(t, tt.want, got, "IsURL(%q)", tt.input)
		})
	}
}
