package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	uimock "github.com/airbytehq/abctl/internal/ui/mock"
)

func TestRenderOutput(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		data          any
		setupMocks    func(ctrl *gomock.Controller) *uimock.MockProvider
		expectedError string
	}{
		{
			name:   "default to JSON",
			format: "",
			data:   map[string]string{"test": "data"},
			setupMocks: func(ctrl *gomock.Controller) *uimock.MockProvider {
				ui := uimock.NewMockProvider(ctrl)
				ui.EXPECT().ShowJSON(map[string]string{"test": "data"}).Return(nil)
				return ui
			},
		},
		{
			name:   "explicit JSON",
			format: "json",
			data:   []string{"item1", "item2"},
			setupMocks: func(ctrl *gomock.Controller) *uimock.MockProvider {
				ui := uimock.NewMockProvider(ctrl)
				ui.EXPECT().ShowJSON([]string{"item1", "item2"}).Return(nil)
				return ui
			},
		},
		{
			name:   "YAML format",
			format: "yaml",
			data:   struct{ Name string }{Name: "test"},
			setupMocks: func(ctrl *gomock.Controller) *uimock.MockProvider {
				ui := uimock.NewMockProvider(ctrl)
				ui.EXPECT().ShowYAML(struct{ Name string }{Name: "test"}).Return(nil)
				return ui
			},
		},
		{
			name:          "unsupported format",
			format:        "xml",
			data:          "test",
			setupMocks:    func(ctrl *gomock.Controller) *uimock.MockProvider { return uimock.NewMockProvider(ctrl) },
			expectedError: "unsupported output format: xml (supported: json, yaml)",
		},
		{
			name:   "JSON error",
			format: "json",
			data:   "test",
			setupMocks: func(ctrl *gomock.Controller) *uimock.MockProvider {
				ui := uimock.NewMockProvider(ctrl)
				ui.EXPECT().ShowJSON("test").Return(errors.New("JSON error"))
				return ui
			},
			expectedError: "JSON error",
		},
		{
			name:   "YAML error",
			format: "yaml",
			data:   "test",
			setupMocks: func(ctrl *gomock.Controller) *uimock.MockProvider {
				ui := uimock.NewMockProvider(ctrl)
				ui.EXPECT().ShowYAML("test").Return(errors.New("YAML error"))
				return ui
			},
			expectedError: "YAML error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ui := tt.setupMocks(ctrl)
			err := RenderOutput(ui, tt.data, tt.format)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}
			assert.NoError(t, err)
		})
	}
}