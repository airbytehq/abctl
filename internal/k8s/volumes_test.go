package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVolumeMounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        []string
		expectMounts []ExtraVolumeMount
		expectErr    error
	}{
		{
			name: "empty input",
		},
		{
			name:         "single valid mount",
			input:        []string{"/host:/container"},
			expectMounts: []ExtraVolumeMount{{HostPath: "/host", ContainerPath: "/container"}},
		},
		{
			name:         "multiple valid mounts",
			input:        []string{"/a:/b", "/c:/d"},
			expectMounts: []ExtraVolumeMount{{HostPath: "/a", ContainerPath: "/b"}, {HostPath: "/c", ContainerPath: "/d"}},
		},
		{
			name:      "invalid spec (missing colon)",
			input:     []string{"/hostcontainer"},
			expectErr: errInvalidVolumeMountSpec("/hostcontainer"),
		},
		{
			name:      "invalid spec (too many colons)",
			input:     []string{"/a:/b:/c"},
			expectErr: errInvalidVolumeMountSpec("/a:/b:/c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mounts, err := ParseVolumeMounts(tt.input)
			assert.Equal(t, tt.expectMounts, mounts, "mounts should match")
			if tt.expectErr != nil {
				assert.EqualError(t, err, tt.expectErr.Error(), "errors should match")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
