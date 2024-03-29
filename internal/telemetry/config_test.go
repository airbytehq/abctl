package telemetry

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/oklog/ulid/v2"
	"os"
	"path/filepath"
	"testing"
)

var id = ulid.Make()

func TestLoadFromFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "analytics-")
	if err != nil {
		t.Fatal("could not create temp file", err)
	}
	defer f.Close()

	if _, err := f.WriteString(`# comments
anonymous_user_id: ` + id.String()); err != nil {
		t.Fatal("could not write to temp file", err)
	}

	cnf, err := LoadFromFile(f.Name())
	if d := cmp.Diff(nil, err); d != "" {
		t.Error("failed to load file", d)
	}
	if d := cmp.Diff(id.String(), cnf.UserID.String()); d != "" {
		t.Error("id is incorrect", d)
	}
}

func TestWriteToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeply", analyticsFile)

	c := Config{UserID: ULID(id)}

	if err := WriteToFile(path, c); err != nil {
		t.Error("failed to create file", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Error("failed to read file", err)
	}

	exp := fmt.Sprintf(`%sanonymous_user_id: %s
`, header, id.String())

	if d := cmp.Diff(exp, string(contents)); d != "" {
		t.Error("contents do not match", d)
	}
}

//func TestULID_MarshalYAML(t *testing.T) {
//	tests := []struct {
//		name    string
//		u       ULID
//		want    any
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := tt.u.MarshalYAML()
//			if (err != nil) != tt.wantErr {
//				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("MarshalYAML() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestULID_UnmarshalYAML(t *testing.T) {
//	type args struct {
//		node *yaml.Node
//	}
//	tests := []struct {
//		name    string
//		u       ULID
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := tt.u.UnmarshalYAML(tt.args.node); (err != nil) != tt.wantErr {
//				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
