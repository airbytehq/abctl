package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/ui"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
)

func TestLogoutCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider)
	}{
		{
			name: "config load error",
			expectedError: "failed to load airbox config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Logging out of Airbyte")
				mockConfig.EXPECT().Load().Return(nil, assert.AnError)

				return mockConfig, mockUI
			},
		},
		{
			name: "not logged in",
			expectedError: "not logged in - no credentials found",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Logging out of Airbyte")
				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: nil,
				}, nil)

				return mockConfig, mockUI
			},
		},
		{
			name: "save config error",
			expectedError: "failed to save config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Logging out of Airbyte")
				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					},
				}, nil)
				mockConfig.EXPECT().Save(gomock.Any()).Return(assert.AnError)

				return mockConfig, mockUI
			},
		},
		{
			name: "success",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigProvider, ui.Provider) {
				mockConfig := airboxmock.NewMockConfigProvider(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				mockUI.EXPECT().Title("Logging out of Airbyte")
				mockConfig.EXPECT().Load().Return(&airbox.Config{
					Credentials: &airbox.Credentials{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					},
				}, nil)
				mockConfig.EXPECT().Save(gomock.Any()).DoAndReturn(func(cfg *airbox.Config) error {
					// Verify credentials were cleared
					assert.Equal(t, "", cfg.Credentials.AccessToken)
					assert.Equal(t, "", cfg.Credentials.RefreshToken)
					return nil
				})
				mockUI.EXPECT().ShowSuccess("Successfully logged out!")
				mockUI.EXPECT().NewLine()

				return mockConfig, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config, ui := tt.setupMocks(ctrl)
			
			cmd := &LogoutCmd{Namespace: "test"}
			err := cmd.Run(context.Background(), config, ui)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
