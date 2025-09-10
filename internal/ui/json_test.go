package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBubbleteaUI_ShowJSON(t *testing.T) {
	tests := []struct {
		name           string
		data           any
		expectedOutput string
		expectedError  string
	}{
		{
			name: "simple struct",
			data: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{
				Name: "test",
				Age:  25,
			},
			expectedOutput: `{
  "name": "test",
  "age": 25
}
`,
		},
		{
			name: "nested struct",
			data: struct {
				User struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"user"`
				Active bool `json:"active"`
			}{
				User: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					ID:   "123",
					Name: "test",
				},
				Active: true,
			},
			expectedOutput: `{
  "user": {
    "id": "123",
    "name": "test"
  },
  "active": true
}
`,
		},
		{
			name: "slice of structs",
			data: []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{
				{ID: 1, Name: "first"},
				{ID: 2, Name: "second"},
			},
			expectedOutput: `[
  {
    "id": 1,
    "name": "first"
  },
  {
    "id": 2,
    "name": "second"
  }
]
`,
		},
		{
			name: "map data",
			data: map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			expectedOutput: `{
  "key1": "value1",
  "key2": 42,
  "key3": true
}
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
			name:          "unmarshalable type",
			data:          make(chan int),
			expectedError: "failed to marshal JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			ui := NewWithOptions(stdout, stderr, nil)

			err := ui.ShowJSON(tt.data)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				// For maps, we need to handle non-deterministic ordering
				if strings.Contains(tt.name, "map") {
					// Just check that all expected lines are present
					actualLines := strings.Split(stdout.String(), "\n")
					expectedLines := strings.Split(tt.expectedOutput, "\n")
					assert.Equal(t, len(expectedLines), len(actualLines))
					for _, line := range expectedLines {
						assert.Contains(t, stdout.String(), strings.TrimSpace(line))
					}
				} else {
					assert.Equal(t, tt.expectedOutput, stdout.String())
				}
				assert.Empty(t, stderr.String())
			}
		})
	}
}
