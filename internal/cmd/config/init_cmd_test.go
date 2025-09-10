package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/airbox/mock"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
)

func TestInitCmd_RunSimplified(t *testing.T) {
	tests := []struct {
		name          string
		force         bool
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider)
		setupEnv      func(t *testing.T)
	}{
		{
			name:  "success cloud config creation",
			force: true,
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(true),
					mockUI.EXPECT().Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"}).Return(0, "Enterprise Flex", nil),
					mockStore.EXPECT().Save(gomock.Any()).Return(nil),
					mockUI.EXPECT().ShowSuccess("Configuration saved successfully!"),
					mockUI.EXPECT().NewLine(),
				)

				return mockStore, mockUI
			},
		},
		{
			name:  "success enterprise config creation",
			force: true,
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(true),
					mockUI.EXPECT().Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"}).Return(1, "Self-Managed Enterprise", nil),
					mockUI.EXPECT().TextInput("Enter your Airbyte instance URL (e.g., https://airbyte.yourcompany.com):", "", nil).Return("https://airbyte.example.com", nil),
					mockStore.EXPECT().Save(gomock.Any()).Return(nil),
					mockUI.EXPECT().ShowSuccess("Configuration saved successfully!"),
					mockUI.EXPECT().NewLine(),
				)

				return mockStore, mockUI
			},
		},
		{
			name:          "config exists without force",
			force:         false,
			expectedError: "airbox configuration already exists, use --force to overwrite",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(true),
				)

				return mockStore, mockUI
			},
		},
		{
			name:          "OAuth config load error cloud",
			force:         false,
			expectedError: "failed to load OAuth config",
			setupEnv: func(t *testing.T) {
				// Don't set OAuth env vars to trigger load error
			},
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(false),
					mockUI.EXPECT().Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"}).Return(0, "Enterprise Flex", nil),
					// No Save() call expected because OAuth config load will fail
				)

				return mockStore, mockUI
			},
		},
		{
			name:          "OAuth config load error enterprise",
			force:         false,
			expectedError: "failed to load OAuth config",
			setupEnv: func(t *testing.T) {
				// Don't set OAuth env vars to trigger load error
			},
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(false),
					mockUI.EXPECT().Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"}).Return(1, "Self-Managed Enterprise", nil),
					mockUI.EXPECT().TextInput("Enter your Airbyte instance URL (e.g., https://airbyte.yourcompany.com):", "", nil).Return("https://airbyte.example.com", nil),
					// No Save() call expected because OAuth config load will fail
				)

				return mockStore, mockUI
			},
		},
		{
			name:          "config save error",
			force:         false,
			expectedError: "failed to save airbox config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(false),
					mockUI.EXPECT().Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"}).Return(0, "Enterprise Flex", nil),
					mockStore.EXPECT().Save(gomock.Any()).Return(errors.New("save failed")),
				)

				return mockStore, mockUI
			},
		},
		{
			name:          "UI select error",
			force:         false,
			expectedError: "failed to get edition",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(false),
					mockUI.EXPECT().Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"}).Return(0, "", errors.New("user cancelled")),
				)

				return mockStore, mockUI
			},
		},
		{
			name:          "enterprise URL input error",
			force:         false,
			expectedError: "failed to get Airbyte URL",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := mock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Initializing airbox configuration"),
					mockStore.EXPECT().Exists().Return(false),
					mockUI.EXPECT().Select("What Airbyte control plane are you connecting to:", []string{"Enterprise Flex", "Self-Managed Enterprise"}).Return(1, "Self-Managed Enterprise", nil),
					mockUI.EXPECT().TextInput("Enter your Airbyte instance URL (e.g., https://airbyte.yourcompany.com):", "", nil).Return("", errors.New("input error")),
				)

				return mockStore, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Cleanup environment variables after test
			t.Cleanup(func() {
				// Ignore any errors.
				_ = os.Unsetenv("AIRBYTE_CLIENT_ID")
				_ = os.Unsetenv("AIRBYTE_CLIENT_SECRET")
			})

			// Set default setupEnv if nil
			if tt.setupEnv == nil {
				tt.setupEnv = func(t *testing.T) {
					t.Setenv("AIRBYTE_CLIENT_ID", "test-client-id")
					t.Setenv("AIRBYTE_CLIENT_SECRET", "test-client-secret")
				}
			}

			// Run environment setup
			tt.setupEnv(t)

			mockStore, mockUI := tt.setupMocks(ctrl)

			cmd := &InitCmd{
				Force: tt.force,
			}

			err := cmd.Run(context.Background(), mockStore, mockUI)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
