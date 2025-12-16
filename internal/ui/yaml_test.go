package ui

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBubbleteaUI_ShowYAML(t *testing.T) {
	tests := []struct {
		name           string
		data           any
		expectedOutput string
		expectedError  string
	}{
		{
			name: "simple struct",
			data: struct {
				Name string `yaml:"name"`
				Age  int    `yaml:"age"`
			}{
				Name: "test",
				Age:  25,
			},
			expectedOutput: `name: test
age: 25
`,
		},
		{
			name: "nested struct",
			data: struct {
				User struct {
					ID   string `yaml:"id"`
					Name string `yaml:"name"`
				} `yaml:"user"`
				Active bool `yaml:"active"`
			}{
				User: struct {
					ID   string `yaml:"id"`
					Name string `yaml:"name"`
				}{
					ID:   "123",
					Name: "test",
				},
				Active: true,
			},
			expectedOutput: `user:
    id: "123"
    name: test
active: true
`,
		},
		{
			name: "slice of structs",
			data: []struct {
				ID   int    `yaml:"id"`
				Name string `yaml:"name"`
			}{
				{ID: 1, Name: "first"},
				{ID: 2, Name: "second"},
			},
			expectedOutput: `- id: 1
  name: first
- id: 2
  name: second
`,
		},
		{
			name: "map data",
			data: map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			expectedOutput: `key1: value1
key2: 42
key3: true
`,
		},
		{
			name:           "nil data",
			data:           nil,
			expectedOutput: "null\n",
		},
		{
			name:           "empty slice",
			data:           []string{},
			expectedOutput: "[]\n",
		},
		{
			name: "string with special characters",
			data: struct {
				Path    string `yaml:"path"`
				Command string `yaml:"command"`
			}{
				Path:    "/usr/local/bin",
				Command: "echo 'hello world'",
			},
			expectedOutput: `path: /usr/local/bin
command: echo 'hello world'
`,
		},
		{
			name: "multi-line string",
			data: struct {
				Description string `yaml:"description"`
			}{
				Description: "This is a\nmulti-line\nstring",
			},
			expectedOutput: `description: |-
    This is a
    multi-line
    string
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			ui := NewWithOptions(stdout, stderr, nil)

			err := ui.ShowYAML(tt.data)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				// For maps with non-deterministic ordering, check line by line
				if _, isMap := tt.data.(map[string]any); isMap {
					actualLines := stdout.String()
					for key := range tt.data.(map[string]any) {
						assert.Contains(t, actualLines, key+":")
					}
				} else {
					assert.Equal(t, tt.expectedOutput, stdout.String())
				}
				assert.Empty(t, stderr.String())
			}
		})
	}
}
