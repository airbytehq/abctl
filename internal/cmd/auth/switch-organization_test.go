package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/http"
	"github.com/airbytehq/abctl/internal/ui"
)

func TestSwitchOrganizationCmd_Run(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		expectedError string
		setupMocks    func(ctrl *gomock.Controller) (http.HTTPDoer, airbox.ConfigProvider, ui.Provider)
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			httpClient, cfg, ui := tt.setupMocks(ctrl)

			cmd := &SwitchOrganizationCmd{}

			err := cmd.Run(context.Background(), httpClient, cfg, ui)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
