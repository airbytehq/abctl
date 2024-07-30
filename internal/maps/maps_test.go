package maps

import (
	"github.com/google/go-cmp/cmp"
	"os"
	"testing"
)

func TestFromSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  map[string]any
	}{
		{
			name:  "empty string",
			input: []string{},
			want:  map[string]any{},
		},
		{
			name:  "single element",
			input: []string{"a=1"},
			want: map[string]any{
				"a": "1",
			},
		},
		{
			name:  "multiple elements",
			input: []string{"a=1", "b=2", "c=3"},
			want: map[string]any{
				"a": "1",
				"b": "2",
				"c": "3",
			},
		},
		{
			name:  "single nested element",
			input: []string{"a.b.c=123"},
			want: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": "123",
					},
				},
			},
		},
		{
			name: "multiple nested elements",
			input: []string{
				"a.b.c=123",
				"a.b.g=127",
				"d.e.f=456",
				"z=26",
			},
			want: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": "123",
						"g": "127",
					},
				},
				"d": map[string]any{
					"e": map[string]any{
						"f": "456",
					},
				},
				"z": "26",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if d := cmp.Diff(tt.want, FromSlice(tt.input)); d != "" {
				t.Error("mismatch (-want, +got) = ", d)
			}
		})
	}
}

func TestFromYMLFile(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want map[string]any
	}{
		{
			name: "empty string",
			want: map[string]any{},
		},
		{
			name: "single element",
			yaml: "a: true",
			want: map[string]any{
				"a": true,
			},
		},
		{
			name: "multiple elements",
			yaml: `a: true
b:
  c: "three"
  d: 4
  e:
    f: 6`,
			want: map[string]any{
				"a": true,
				"b": map[string]any{
					"c": "three",
					"d": 4,
					"e": map[string]any{
						"f": 6,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "*.yml")
			if err != nil {
				t.Fatal("could not create temp file", err)
			}
			if _, err := f.WriteString(tt.yaml); err != nil {
				t.Fatal("could not write to temp file", err)
			}
			_ = f.Close()

			m, err := FromYAMLFile(f.Name())
			if err != nil {
				t.Fatal("could not read from maps", err)
			}
			if d := cmp.Diff(tt.want, m); d != "" {
				t.Error("mismatch (-want, +got) = ", d)
			}
		})
	}

	t.Run("no file provided", func(t *testing.T) {
		m, err := FromYAMLFile("")
		if err != nil {
			t.Fatal("could not read from maps", err)
		}
		// if no file is provided, any empty map is returned
		if d := cmp.Diff(map[string]any{}, m); d != "" {
			t.Error("mismatch (-want, +got) = ", d)
		}
	})
}

func TestToYAML(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		want string
	}{
		{
			name: "empty map",
			m:    map[string]any{},
			want: "{}\n",
		},
		{
			name: "single element",
			m: map[string]any{
				"a": 1,
			},
			want: "a: 1\n",
		},
		{
			name: "multiple elements",
			m: map[string]any{
				"a": 1,
				"b": "2",
				"c": "three",
			},
			want: `a: 1
b: "2"
c: three
`,
		},
		{
			name: "nested elements",
			m: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": 123,
						"d": "124",
					},
				},
				"z": "26",
			},
			want: `a:
    b:
        c: 123
        d: "124"
z: "26"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			have, err := ToYAML(tt.m)
			if err != nil {
				t.Fatal("could not convert maps to YAML", err)
			}
			if d := cmp.Diff(tt.want, have); d != "" {
				t.Error("mismatch (-want, +got) = ", d)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		base map[string]any
		over map[string]any
		want map[string]any
	}{
		{
			name: "empty",
			base: map[string]any{},
			over: map[string]any{},
			want: map[string]any{},
		},
		{
			name: "single element base only",
			base: map[string]any{"a": "1"},
			over: map[string]any{},
			want: map[string]any{"a": "1"},
		},
		{
			name: "single element over only",
			base: map[string]any{},
			over: map[string]any{"a": "1"},
			want: map[string]any{"a": "1"},
		},
		{
			name: "single element same in base and over",
			base: map[string]any{"a": "1"},
			over: map[string]any{"a": "26"},
			want: map[string]any{"a": "26"},
		},
		{
			name: "single element, different type in base",
			base: map[string]any{"a": 1},
			over: map[string]any{"a": "26"},
			want: map[string]any{"a": "26"},
		},
		{
			name: "single element, different type in over",
			base: map[string]any{"a": "1"},
			over: map[string]any{"a": 26},
			want: map[string]any{"a": 26},
		},
		{
			name: "single element diff in base and over",
			base: map[string]any{"a": "1"},
			over: map[string]any{"z": "26"},
			want: map[string]any{
				"a": "1",
				"z": "26",
			},
		},
		{
			name: "singled element diff in nested base",
			base: map[string]any{
				"a": "1",
				"b": map[string]any{
					"c": true,
				},
			},
			over: map[string]any{
				"b": map[string]any{
					"c": false,
					"d": "100",
				},
				"z": "26",
			},
			want: map[string]any{
				"a": "1",
				"b": map[string]any{
					"c": false,
					"d": "100",
				},
				"z": "26",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Merge(tt.base, tt.over)
			if d := cmp.Diff(tt.want, tt.base); d != "" {
				t.Error("mismatch (-want, +got) = ", d)
			}
		})
	}
}
