package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/airbytehq/abctl/internal/airbox"
	airboxmock "github.com/airbytehq/abctl/internal/airbox/mock"
	"github.com/airbytehq/abctl/internal/auth"
	uimock "github.com/airbytehq/abctl/internal/ui/mock"
)

func TestLogoutCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider)
	}{
		{
			name: "success logout",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Logging out of Airbyte"),
					mockStore.EXPECT().Load().Return(&airbox.Config{
						Credentials: &auth.Credentials{
							AccessToken: "test-token",
							TokenType:   "Bearer",
							ExpiresAt:   time.Now().Add(time.Hour),
						},
					}, nil),
					mockStore.EXPECT().Save(gomock.Any()).Return(nil),
					mockUI.EXPECT().ShowSuccess("Successfully logged out!"),
					mockUI.EXPECT().NewLine(),
				)

				return mockStore, mockUI
			},
		},
		{
			name:          "config save error",
			expectedError: "failed to save config",
			setupMocks: func(ctrl *gomock.Controller) (airbox.ConfigStore, *uimock.MockProvider) {
				mockStore := airboxmock.NewMockConfigStore(ctrl)
				mockUI := uimock.NewMockProvider(ctrl)

				gomock.InOrder(
					mockUI.EXPECT().Title("Logging out of Airbyte"),
					mockStore.EXPECT().Load().Return(&airbox.Config{
						Credentials: &auth.Credentials{
							AccessToken: "test-token",
						},
					}, nil),
					mockStore.EXPECT().Save(gomock.Any()).Return(assert.AnError),
				)

				return mockStore, mockUI
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore, mockUI := tt.setupMocks(ctrl)

			cmd := &LogoutCmd{}
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
