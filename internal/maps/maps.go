package maps

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FromSlice converts a slice of dot-delimited string values into a map[string]any.
// For example:
// "a.b.c=123","a.b.d=124" would return { "a": { "b": { "c": 123, "d": 124 } } }
func FromSlice(slice []string) map[string]any {
	m := map[string]any{}

	for _, s := range slice {
		// s has the format of:
		// a.b.c=xyz
		parts := strings.SplitN(s, "=", 2)
		// a.b.c becomes [a, b, c]
		keys := strings.Split(parts[0], ".")
		// xyz
		value := parts[1]

		// pointer to the root of the map,
		// as this map will contain nested maps, this pointer will be
		// updated to point to the root of the nested maps as it iterates
		// through the for loop
		p := m
		for i, k := range keys {
			// last key, put the value into the map
			if i == len(keys)-1 {
				p[k] = value
				continue
			}
			// if the nested map doesn't exist, create it
			if _, ok := p[k]; !ok {
				p[k] = map[string]any{}
			}
			// cast the key to a map[string]any
			// and update the pointer to point to it
			p = p[k].(map[string]any)
		}
	}

	return m
}

// FromYAMLFile converts a yaml file into a map[string]any.
func FromYAMLFile(path string) (map[string]any, error) {
	if path == "" {
		return map[string]any{}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s: %w", path, err)
	}
	// ensure we don't return `nil, nil`
	if m == nil {
		return map[string]any{}, nil
	}
	return m, nil
}

// ToYAML converts the m map into a yaml string.
// E.g. map[string]any{"a" : 1, "b", 2} becomes
// a: 1
// b: 2
func ToYAML(m map[string]any) (string, error) {
	raw, err := yaml.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to marshal map: %w", err)
	}
	return string(raw), nil
}

// Merge merges the override map into the base map.
// Modifying the base map in place.
func Merge(base, override map[string]any) {
	for k, overrideVal := range override {
		if baseVal, ok := base[k]; ok {
			// both maps have this key
			baseChild, baseChildIsMap := baseVal.(map[string]any)
			overrideChild, overrideChildIsMap := overrideVal.(map[string]any)

			if baseChildIsMap && overrideChildIsMap {
				// both values are maps, recurse
				Merge(baseChild, overrideChild)
			} else {
				// override base with override
				base[k] = overrideVal
			}
		} else {
			// only override has this key
			base[k] = overrideVal
		}
	}
}
