package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/ui"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInitCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		cmd           *InitCmd
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider)
		configSetup   func(t *testing.T)
	}{
		{
			name: "namespace from context",
			cmd: &InitCmd{
				Force: false,
			},
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(0, "Email/Password", nil).Times(1)
				mockUI.EXPECT().ShowSection("Configuration:", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
				mockUI.EXPECT().ShowInfo("Configuration saved to local airbox config!").Times(1)
				mockUI.EXPECT().ShowKeyValue("Config file", gomock.Any()).Times(1)
				mockUI.EXPECT().NewLine().Times(1)

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				// Mock LoadConfig to return empty config (for writes)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
		{
			name: "configmap exists with force flag",
			cmd: &InitCmd{
				Force: true,
			},
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(0, "Email/Password", nil).Times(1)
				mockUI.EXPECT().ShowSection("Configuration:", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
				mockUI.EXPECT().ShowInfo("Configuration saved to local airbox config!").Times(1)
				mockUI.EXPECT().ShowKeyValue("Config file", gomock.Any()).Times(1)
				mockUI.EXPECT().NewLine().Times(1)

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				// Mock LoadConfig to return empty config (for writes)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
		{
			name: "enterprise invalid endpoint",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "invalid enterprise endpoint",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				// Mock UI interactions for enterprise SME setup with invalid endpoint
				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(0, "Enterprise (SME)", nil).Times(1)
				mockUI.EXPECT().TextInput("Enter your SME endpoint URL:", "https://your-sme.company.com", gomock.Any()).
					Return("https://invalid-sme.company.com", nil).Times(1)

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
		{
			name: "enterprise endpoint input cancelled",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "failed to get SME endpoint",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(0, "Enterprise (SME)", nil).Times(1)
				mockUI.EXPECT().TextInput("Enter your SME endpoint URL:", "https://your-sme.company.com", gomock.Any()).
					Return("", errors.New("cancelled")).Times(1)

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
		{
			name: "deployment type selection cancelled",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "failed to select deployment type",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(0, "", errors.New("cancelled")).Times(1)
				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				// Mock LoadConfig to return empty config (for writes)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
		{
			name: "cloud auth method selection cancelled",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "failed to select auth method",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(0, "", errors.New("cancelled")).Times(1)
				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				// Mock LoadConfig to return empty config (for writes)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
		{
			name: "SSO identifier cancelled",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "failed to get company identifier",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(1, "SSO (Single Sign-On)", nil).Times(1)
				mockUI.EXPECT().TextInput("Enter your company identifier (from your SSO login URL):", "my-company", gomock.Any()).
					Return("", errors.New("cancelled")).Times(1)
				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				// Mock LoadConfig to return empty config (for writes)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
		{
			name: "config already exists without force",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "airbox configuration already exists, use --force to overwrite",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(0, "Email/Password", nil).Times(1)
				mockUI.EXPECT().ShowSection("Configuration:", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				// Return existing config to trigger the error
				configProvider.EXPECT().Load().Return(&airbox.Config{
					Contexts: []airbox.NamedContext{{Name: "existing"}},
				}, nil).Times(1)

				return configProvider, mockUI
			},
		},
		{
			name: "config save error",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "failed to save airbox config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(0, "Email/Password", nil).Times(1)
				mockUI.EXPECT().ShowSection("Configuration:", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).Times(1)
				configProvider.EXPECT().Save(gomock.Any()).Return(errors.New("save failed")).Times(1)

				return configProvider, mockUI
			},
		},
		{
			name: "config load error",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "failed to load airbox config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(0, "Email/Password", nil).Times(1)
				mockUI.EXPECT().ShowSection("Configuration:", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().Load().Return(nil, errors.New("load failed")).Times(1)

				return configProvider, mockUI
			},
		},
		{
			name: "cloud config build error",
			cmd: &InitCmd{
				Force: false,
			},
			expectedError: "failed to build cloud config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Initializing airbox configuration").Times(1)
				mockUI.EXPECT().Select("Select your Airbyte deployment type:", []string{"Enterprise (SME)", "Cloud"}).
					Return(1, "Cloud", nil).Times(1)
				mockUI.EXPECT().Select("How do you sign in to Airbyte Cloud?", []string{"Email/Password", "SSO (Single Sign-On)"}).
					Return(1, "SSO (Single Sign-On)", nil).Times(1)
				mockUI.EXPECT().TextInput("Enter your company identifier (from your SSO login URL):", "my-company", gomock.Any()).
					Return("", nil).Times(1) // Empty company ID will cause build error

				configProvider := airboxmock.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().Load().Return(&airbox.Config{}, nil).AnyTimes()
				configProvider.EXPECT().Save(gomock.Any()).Return(nil).AnyTimes()

				return configProvider, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Setup test config if provided, otherwise use clean environment
			if tt.configSetup != nil {
				tt.configSetup(t)
			} else {
				// Default: clean config environment
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config")
				os.Setenv("AIRBOXCONFIG", configPath)
			}

			configProvider, uiProvider := tt.setupMocks(ctrl)

			// Execute
			err := tt.cmd.Run(context.Background(), configProvider, uiProvider)

			// Assert
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
