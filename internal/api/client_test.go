package api

import (
	"errors"
	"testing"

	"github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// errorReader simulates io.ReadAll failure for testing
type errorReader struct{}

// Read just simulates an error return
func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// Close the error reader
func (e *errorReader) Close() error {
	return nil
}

func TestNewClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDoer := mock.NewMockHTTPDoer(ctrl)
	client := NewClient(mockDoer)

	assert.NotNil(t, client)
	assert.Equal(t, mockDoer, client.http)
}
