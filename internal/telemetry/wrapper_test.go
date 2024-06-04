package telemetry

import (
	"bytes"
	"context"
	"errors"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"os"
	"strings"
	"testing"
)

var origInstance = instance

func TestWrapper(t *testing.T) {
	t.Cleanup(func() {
		instance = origInstance
	})

	tests := []struct {
		name       string
		err        error
		expStart   bool
		expSuccess bool
		expFailure bool
	}{
		{
			name:       "success",
			expStart:   true,
			expSuccess: true,
		},
		{
			name:       "failure",
			err:        errors.New("should fail"),
			expStart:   true,
			expFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startCalled := false
			successCalled := false
			failureCalled := false

			instance = MockClient{
				start: func(ctx context.Context, eventType EventType) error {
					startCalled = true
					return nil
				},
				success: func(ctx context.Context, eventType EventType) error {
					successCalled = true
					return nil
				},
				failure: func(ctx context.Context, eventType EventType, err error) error {
					failureCalled = true
					return nil
				},
			}

			f := func() error {
				return tt.err
			}

			err := Wrapper(context.Background(), Install, f)

			if d := cmp.Diff(tt.err, err, cmpopts.EquateErrors()); d != "" {
				t.Errorf("error mismatch (-want, +got): %s", d)
			}
			if d := cmp.Diff(tt.expStart, startCalled); d != "" {
				t.Errorf("Start() called (-want +got):\n%s", d)
			}
			if d := cmp.Diff(tt.expSuccess, successCalled); d != "" {
				t.Errorf("Success() called (-want +got):\n%s", d)
			}
			if d := cmp.Diff(tt.expFailure, failureCalled); d != "" {
				t.Errorf("Failure() called (-want +got):\n%s", d)
			}
		})
	}
}

// TestWrapper_DebugLogs verifies that the messages output by the Wrapper function are only visible
// if debug messages are enabled.
func TestWrapper_DebugLogs(t *testing.T) {
	b := bytes.NewBufferString("")
	pterm.SetDefaultOutput(b)
	t.Cleanup(func() {
		instance = origInstance
		pterm.DisableDebugMessages()
		pterm.SetDefaultOutput(os.Stdout)
	})

	pterm.EnableDebugMessages()

	tests := []struct {
		name       string
		err        error
		errStart   error
		expStart   bool
		errSuccess error
		expSuccess bool
		errFailure error
		expFailure bool
	}{
		{
			name:     "start",
			errStart: errors.New("start"),
			expStart: true,
		},
		{
			name:       "success",
			errSuccess: errors.New("success"),
			expStart:   true,
			expSuccess: true,
		},
		{
			name:       "failure",
			err:        errors.New("wrapped error"),
			errFailure: errors.New("failure"),
			expStart:   true,
			expFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() { b.Reset() })

			startCalled := false
			successCalled := false
			failureCalled := false

			instance = MockClient{
				start: func(ctx context.Context, eventType EventType) error {
					startCalled = true
					return tt.errStart
				},
				success: func(ctx context.Context, eventType EventType) error {
					successCalled = true
					return tt.errSuccess
				},
				failure: func(ctx context.Context, eventType EventType, err error) error {
					failureCalled = true
					return tt.errFailure
				},
			}

			f := func() error {
				return tt.err
			}

			err := Wrapper(context.Background(), Install, f)

			if d := cmp.Diff(tt.err, err, cmpopts.EquateErrors()); d != "" {
				t.Errorf("error mismatch (-want, +got): %s", d)
			}
			if d := cmp.Diff(tt.expStart, startCalled); d != "" {
				t.Errorf("Start() called (-want +got):\n%s", d)
			}
			if d := cmp.Diff(tt.expSuccess, successCalled); d != "" {
				t.Errorf("Success() called (-want +got):\n%s", d)
			}
			if d := cmp.Diff(tt.expFailure, failureCalled); d != "" {
				t.Errorf("Failure() called (-want +got):\n%s", d)
			}

			output := b.String()
			if tt.errStart != nil {
				if !strings.Contains(output, "DEBUG") {
					t.Errorf("Start() does not contain DEBUG: %s", output)
				}
				if !strings.Contains(output, tt.errStart.Error()) {
					t.Errorf("Start() does not contain expected error: %s", output)
				}
			}
			if tt.errSuccess != nil {
				if !strings.Contains(output, "DEBUG") {
					t.Errorf("Success() does not contain DEBUG: %s", output)
				}
				if !strings.Contains(output, tt.errSuccess.Error()) {
					t.Errorf("Success() does not contain expected error: %s", output)
				}
			}
			if tt.errFailure != nil {
				if !strings.Contains(output, "DEBUG") {
					t.Errorf("Failure() does not contain DEBUG: %s", output)
				}
				if !strings.Contains(output, tt.errFailure.Error()) {
					t.Errorf("Failure() does not contain expected error: %s", output)
				}
			}
		})
	}
}

var _ Client = (*MockClient)(nil)

type MockClient struct {
	start   func(ctx context.Context, eventType EventType) error
	success func(ctx context.Context, eventType EventType) error
	failure func(ctx context.Context, eventType EventType, err error) error
	attr    func(key, val string)
	user    func() uuid.UUID
}

func (m MockClient) Start(ctx context.Context, eventType EventType) error {
	return m.start(ctx, eventType)
}

func (m MockClient) Success(ctx context.Context, eventType EventType) error {
	return m.success(ctx, eventType)
}

func (m MockClient) Failure(ctx context.Context, eventType EventType, err error) error {
	return m.failure(ctx, eventType, err)
}

func (m MockClient) Attr(key, val string) {
	m.attr(key, val)
}

func (m MockClient) User() uuid.UUID {
	return m.user()
}
