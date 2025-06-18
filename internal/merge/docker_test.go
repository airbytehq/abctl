package merge

import (
	"reflect"
	"testing"
)

func TestDockerImages(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{
			name: "empty lists",
			a:    []string{},
			b:    []string{},
			want: []string{},
		},
		{
			name: "only list a",
			a:    []string{"nginx:1.20", "postgres:13"},
			b:    []string{},
			want: []string{"nginx:1.20", "postgres:13"},
		},
		{
			name: "only list b",
			a:    []string{},
			b:    []string{"redis:6", "mongo:5"},
			want: []string{"mongo:5", "redis:6"},
		},
		{
			name: "merge without conflicts",
			a:    []string{"nginx:1.20", "postgres:13"},
			b:    []string{"redis:6", "mongo:5"},
			want: []string{"mongo:5", "nginx:1.20", "postgres:13", "redis:6"},
		},
		{
			name: "override with conflicts",
			a:    []string{"nginx:1.20", "postgres:13", "redis:6"},
			b:    []string{"postgres:14", "mongo:5"},
			want: []string{"mongo:5", "nginx:1.20", "postgres:14", "redis:6"},
		},
		{
			name: "images without tags default to latest",
			a:    []string{"nginx", "postgres:13"},
			b:    []string{"redis", "mongo:5"},
			want: []string{"mongo:5", "nginx:latest", "postgres:13", "redis:latest"},
		},
		{
			name: "override image with explicit tag to latest",
			a:    []string{"nginx:1.20"},
			b:    []string{"nginx"},
			want: []string{"nginx:latest"},
		},
		{
			name: "override image with latest to explicit tag",
			a:    []string{"nginx"},
			b:    []string{"nginx:1.21"},
			want: []string{"nginx:1.21"},
		},
		{
			name: "complex registry paths",
			a:    []string{"docker.io/library/nginx:1.20", "gcr.io/project/postgres:13"},
			b:    []string{"docker.io/library/nginx:1.21", "quay.io/redis:6"},
			want: []string{"docker.io/library/nginx:1.21", "gcr.io/project/postgres:13", "quay.io/redis:6"},
		},
		{
			name: "same image multiple times in b",
			a:    []string{"nginx:1.20"},
			b:    []string{"nginx:1.21", "nginx:1.22"},
			want: []string{"nginx:1.22"}, // last one wins
		},
		{
			name: "duplicate images in list a",
			a:    []string{"nginx:1.20", "nginx:1.21"},
			b:    []string{},
			want: []string{"nginx:1.21"}, // last one wins
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DockerImages(tt.a, tt.b)

			// Handle nil vs empty slice comparison
			if len(got) == 0 && len(tt.want) == 0 {
				return // both empty, test passes
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DockerImages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDockerImage(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		wantRepo string
		wantTag  string
	}{
		{
			name:     "image with tag",
			image:    "nginx:1.20",
			wantRepo: "nginx",
			wantTag:  "1.20",
		},
		{
			name:     "image without tag",
			image:    "nginx",
			wantRepo: "nginx",
			wantTag:  "latest",
		},
		{
			name:     "image with registry and tag",
			image:    "docker.io/library/nginx:1.20",
			wantRepo: "docker.io/library/nginx",
			wantTag:  "1.20",
		},
		{
			name:     "image with registry without tag",
			image:    "gcr.io/project/postgres",
			wantRepo: "gcr.io/project/postgres",
			wantTag:  "latest",
		},
		{
			name:     "image with port in registry",
			image:    "localhost:5000/myimage:v1.0",
			wantRepo: "localhost:5000/myimage",
			wantTag:  "v1.0",
		},
		{
			name:     "image with multiple colons",
			image:    "registry.com:5000/ns/image:tag:with:colons",
			wantRepo: "registry.com:5000/ns/image",
			wantTag:  "tag:with:colons",
		},
		{
			name:     "empty string",
			image:    "",
			wantRepo: "",
			wantTag:  "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepo, gotTag := parseDockerImage(tt.image)
			if gotRepo != tt.wantRepo {
				t.Errorf("parseDockerImage() repo = %v, want %v", gotRepo, tt.wantRepo)
			}
			if gotTag != tt.wantTag {
				t.Errorf("parseDockerImage() tag = %v, want %v", gotTag, tt.wantTag)
			}
		})
	}
}
