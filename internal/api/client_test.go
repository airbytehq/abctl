package api

import (
	"testing"

	"github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDoer := mock.NewMockHTTPDoer(ctrl)
	client := NewClient(mockDoer)

	assert.NotNil(t, client)
	assert.Equal(t, mockDoer, client.http)
}
