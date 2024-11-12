package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/pbnjay/memory"
)

var userID = uuid.New()
var sessionID = uuid.New()

func TestSegmentClient_Options(t *testing.T) {
	mDoer := &mockDoer{}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{}, opts...)

	if d := cmp.Diff(sessionID, cli.sessionID); d != "" {
		t.Error("sessionID mismatch (-want +got):", d)
	}
	if d := cmp.Diff(mDoer, cli.doer, cmp.AllowUnexported(mockDoer{})); d != "" {
		t.Error("doer mismatch (-want +got):", d)
	}
}

func TestSegmentClient_Start(t *testing.T) {
	var req *http.Request
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			req = r
			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()

	if err := cli.Start(ctx, Install); err != nil {
		t.Error("start call failed", err)
	}

	// url
	if d := cmp.Diff(url, req.URL.String()); d != "" {
		t.Error("request URL mismatch (-want +got):", d)
	}
	// method
	if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
		t.Error("request method mismatch (-want +got):", d)
	}
	// content-type
	if d := cmp.Diff("application/json", req.Header.Get("Content-Type")); d != "" {
		t.Error("request header mismatch (-want +got):", d)
	}
	// body
	reqBodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Error("unable to read request body", err)
	}
	var reqBody body
	if err := json.Unmarshal(reqBodyRaw, &reqBody); err != nil {
		t.Error("unable to unmarshal request body", err)
	}

	if d := cmp.Diff(userID.String(), reqBody.ID); d != "" {
		t.Error("request ID mismatch (-want +got):", d)
	}

	if d := cmp.Diff(string(Install), reqBody.Event); d != "" {
		t.Error("request event mismatch (-want +got):", d)
	}

	if d := cmp.Diff(time.Now().UTC().Format(time.RFC3339), reqBody.Timestamp, cmpopts.EquateApproxTime(1*time.Second)); d != "" {
		t.Error("request timestamp mismatch (-want +got):", d)
	}

	if d := cmp.Diff(trackingKey, reqBody.WriteKey); d != "" {
		t.Error("request tracking key mismatch (-want +got):", d)
	}
	// body properties
	if d := cmp.Diff(9, len(reqBody.Properties)); d != "" {
		t.Error("request property count mismatch (-want +got):", d)
	}
	if d := cmp.Diff("abctl", reqBody.Properties["deployment_method"]); d != "" {
		t.Error("request deployment_method mismatch (-want +got):", d)
	}
	if d := cmp.Diff(sessionID.String(), reqBody.Properties["session_id"]); d != "" {
		t.Error("request session_id mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Start), reqBody.Properties["state"]); d != "" {
		t.Error("request state mismatch (-want +got):", d)
	}
	if d := cmp.Diff(runtime.GOOS, reqBody.Properties["os"]); d != "" {
		t.Error("request os mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["build"]); d != "" {
		t.Error("request build mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["script_version"]); d != "" {
		t.Error("request script_version mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.Itoa(runtime.NumCPU()), reqBody.Properties["cpu_count"]); d != "" {
		t.Error("request cpu_count mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.FormatUint(memory.TotalMemory(), 10), reqBody.Properties["mem_total_bytes"]); d != "" {
		t.Error("request mem_total_bytes mismatch (-want +got):", d)
	}
	// free memory will fluctuate, only check it has a value greater than 0
	if v, ok := reqBody.Properties["mem_free_bytes"]; !ok {
		t.Error("request mem_free_bytes is missing")
	} else {
		free, _ := strconv.Atoi(v)
		if free <= 0 {
			t.Error("request mem_free_bytes is not set", v)
		}
	}
	// error should not be set
	if _, ok := reqBody.Properties["error"]; ok {
		t.Error("request error is present")
	}
}

func TestSegmentClient_StartWithAttr(t *testing.T) {
	var req *http.Request
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			req = r
			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)
	cli.Attr("key1", "val1")
	cli.Attr("key2", "val2")

	ctx := context.Background()

	if err := cli.Start(ctx, Install); err != nil {
		t.Error("start call failed", err)
	}

	// url
	if d := cmp.Diff(url, req.URL.String()); d != "" {
		t.Error("request URL mismatch (-want +got):", d)
	}
	// method
	if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
		t.Error("request method mismatch (-want +got):", d)
	}
	// content-type
	if d := cmp.Diff("application/json", req.Header.Get("Content-Type")); d != "" {
		t.Error("request header mismatch (-want +got):", d)
	}
	// body
	reqBodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Error("unable to read request body", err)
	}
	var reqBody body
	if err := json.Unmarshal(reqBodyRaw, &reqBody); err != nil {
		t.Error("unable to unmarshal request body", err)
	}

	if d := cmp.Diff(userID.String(), reqBody.ID); d != "" {
		t.Error("request ID mismatch (-want +got):", d)
	}

	if d := cmp.Diff(string(Install), reqBody.Event); d != "" {
		t.Error("request event mismatch (-want +got):", d)
	}

	if d := cmp.Diff(time.Now().UTC().Format(time.RFC3339), reqBody.Timestamp, cmpopts.EquateApproxTime(1*time.Second)); d != "" {
		t.Error("request timestamp mismatch (-want +got):", d)
	}

	if d := cmp.Diff(trackingKey, reqBody.WriteKey); d != "" {
		t.Error("request tracking key mismatch (-want +got):", d)
	}
	// body properties
	if d := cmp.Diff(11, len(reqBody.Properties)); d != "" {
		t.Error("request property count mismatch (-want +got):", d)
	}
	if d := cmp.Diff("abctl", reqBody.Properties["deployment_method"]); d != "" {
		t.Error("request deployment_method mismatch (-want +got):", d)
	}
	if d := cmp.Diff(sessionID.String(), reqBody.Properties["session_id"]); d != "" {
		t.Error("request session_id mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Start), reqBody.Properties["state"]); d != "" {
		t.Error("request state mismatch (-want +got):", d)
	}
	if d := cmp.Diff(runtime.GOOS, reqBody.Properties["os"]); d != "" {
		t.Error("request os mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["build"]); d != "" {
		t.Error("request build mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["script_version"]); d != "" {
		t.Error("request script_version mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.Itoa(runtime.NumCPU()), reqBody.Properties["cpu_count"]); d != "" {
		t.Error("request cpu_count mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.FormatUint(memory.TotalMemory(), 10), reqBody.Properties["mem_total_bytes"]); d != "" {
		t.Error("request mem_total_bytes mismatch (-want +got):", d)
	}
	// free memory will fluctuate, only check it has a value greater than 0
	if v, ok := reqBody.Properties["mem_free_bytes"]; !ok {
		t.Error("request mem_free_bytes is missing")
	} else {
		free, _ := strconv.Atoi(v)
		if free <= 0 {
			t.Error("request mem_free_bytes is not set", v)
		}
	}
	if d := cmp.Diff("val1", reqBody.Properties["key1"]); d != "" {
		t.Error("request key1 mismatch (-want +got):", d)
	}
	if d := cmp.Diff("val2", reqBody.Properties["key2"]); d != "" {
		t.Error("request key1 mismatch (-want +got):", d)
	}
	// error should not be set
	if _, ok := reqBody.Properties["error"]; ok {
		t.Error("request error is present")
	}
}

func TestSegmentClient_StartErr(t *testing.T) {
	httpErr := errors.New("http error")
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			return nil, httpErr
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()

	if err := cli.Start(ctx, Install); err == nil {
		t.Error("start call should have failed")
	} else if !errors.Is(err, httpErr) {
		t.Error("start call error should contain http error", err)
	}
}

func TestSegmentClient_Success(t *testing.T) {
	var req *http.Request
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			req = r
			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()

	if err := cli.Success(ctx, Install); err != nil {
		t.Error("start call failed", err)
	}

	// url
	if d := cmp.Diff(url, req.URL.String()); d != "" {
		t.Error("request URL mismatch (-want +got):", d)
	}
	// method
	if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
		t.Error("request method mismatch (-want +got):", d)
	}
	// content-type
	if d := cmp.Diff("application/json", req.Header.Get("Content-Type")); d != "" {
		t.Error("request header mismatch (-want +got):", d)
	}
	// body
	reqBodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Error("unable to read request body", err)
	}
	var reqBody body
	if err := json.Unmarshal(reqBodyRaw, &reqBody); err != nil {
		t.Error("unable to unmarshal request body", err)
	}

	if d := cmp.Diff(userID.String(), reqBody.ID); d != "" {
		t.Error("request ID mismatch (-want +got):", d)
	}

	if d := cmp.Diff(string(Install), reqBody.Event); d != "" {
		t.Error("request event mismatch (-want +got):", d)
	}

	if d := cmp.Diff(time.Now().UTC().Format(time.RFC3339), reqBody.Timestamp, cmpopts.EquateApproxTime(1*time.Second)); d != "" {
		t.Error("request timestamp mismatch (-want +got):", d)
	}

	if d := cmp.Diff(trackingKey, reqBody.WriteKey); d != "" {
		t.Error("request tracking key mismatch (-want +got):", d)
	}
	// body properties
	if d := cmp.Diff(9, len(reqBody.Properties)); d != "" {
		t.Error("request property count mismatch (-want +got):", d)
	}
	if d := cmp.Diff("abctl", reqBody.Properties["deployment_method"]); d != "" {
		t.Error("request deployment_method mismatch (-want +got):", d)
	}
	if d := cmp.Diff(sessionID.String(), reqBody.Properties["session_id"]); d != "" {
		t.Error("request session_id mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Success), reqBody.Properties["state"]); d != "" {
		t.Error("request state mismatch (-want +got):", d)
	}
	if d := cmp.Diff(runtime.GOOS, reqBody.Properties["os"]); d != "" {
		t.Error("request os mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["build"]); d != "" {
		t.Error("request build mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["script_version"]); d != "" {
		t.Error("request script_version mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.Itoa(runtime.NumCPU()), reqBody.Properties["cpu_count"]); d != "" {
		t.Error("request cpu_count mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.FormatUint(memory.TotalMemory(), 10), reqBody.Properties["mem_total_bytes"]); d != "" {
		t.Error("request mem_total_bytes mismatch (-want +got):", d)
	}
	// free memory will fluctuate, only check it has a value greater than 0
	if v, ok := reqBody.Properties["mem_free_bytes"]; !ok {
		t.Error("request mem_free_bytes is missing")
	} else {
		free, _ := strconv.Atoi(v)
		if free <= 0 {
			t.Error("request mem_free_bytes is not set", v)
		}
	}
	// error should not be set
	if _, ok := reqBody.Properties["error"]; ok {
		t.Error("request error is present")
	}
}

func TestSegmentClient_SuccessWithAttr(t *testing.T) {
	var req *http.Request
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			req = r
			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)
	cli.Attr("key1", "val1")
	cli.Attr("key2", "val2")

	ctx := context.Background()

	if err := cli.Success(ctx, Install); err != nil {
		t.Error("start call failed", err)
	}

	// url
	if d := cmp.Diff(url, req.URL.String()); d != "" {
		t.Error("request URL mismatch (-want +got):", d)
	}
	// method
	if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
		t.Error("request method mismatch (-want +got):", d)
	}
	// content-type
	if d := cmp.Diff("application/json", req.Header.Get("Content-Type")); d != "" {
		t.Error("request header mismatch (-want +got):", d)
	}
	// body
	reqBodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Error("unable to read request body", err)
	}
	var reqBody body
	if err := json.Unmarshal(reqBodyRaw, &reqBody); err != nil {
		t.Error("unable to unmarshal request body", err)
	}

	if d := cmp.Diff(userID.String(), reqBody.ID); d != "" {
		t.Error("request ID mismatch (-want +got):", d)
	}

	if d := cmp.Diff(string(Install), reqBody.Event); d != "" {
		t.Error("request event mismatch (-want +got):", d)
	}

	if d := cmp.Diff(time.Now().UTC().Format(time.RFC3339), reqBody.Timestamp, cmpopts.EquateApproxTime(1*time.Second)); d != "" {
		t.Error("request timestamp mismatch (-want +got):", d)
	}

	if d := cmp.Diff(trackingKey, reqBody.WriteKey); d != "" {
		t.Error("request tracking key mismatch (-want +got):", d)
	}
	// body properties
	if d := cmp.Diff(11, len(reqBody.Properties)); d != "" {
		t.Error("request property count mismatch (-want +got):", d)
	}
	if d := cmp.Diff("abctl", reqBody.Properties["deployment_method"]); d != "" {
		t.Error("request deployment_method mismatch (-want +got):", d)
	}
	if d := cmp.Diff(sessionID.String(), reqBody.Properties["session_id"]); d != "" {
		t.Error("request session_id mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Success), reqBody.Properties["state"]); d != "" {
		t.Error("request state mismatch (-want +got):", d)
	}
	if d := cmp.Diff(runtime.GOOS, reqBody.Properties["os"]); d != "" {
		t.Error("request os mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["build"]); d != "" {
		t.Error("request build mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["script_version"]); d != "" {
		t.Error("request script_version mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.Itoa(runtime.NumCPU()), reqBody.Properties["cpu_count"]); d != "" {
		t.Error("request cpu_count mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.FormatUint(memory.TotalMemory(), 10), reqBody.Properties["mem_total_bytes"]); d != "" {
		t.Error("request mem_total_bytes mismatch (-want +got):", d)
	}
	// free memory will fluctuate, only check it has a value greater than 0
	if v, ok := reqBody.Properties["mem_free_bytes"]; !ok {
		t.Error("request mem_free_bytes is missing")
	} else {
		free, _ := strconv.Atoi(v)
		if free <= 0 {
			t.Error("request mem_free_bytes is not set", v)
		}
	}
	if d := cmp.Diff("val1", reqBody.Properties["key1"]); d != "" {
		t.Error("request key1 mismatch (-want +got):", d)
	}
	if d := cmp.Diff("val2", reqBody.Properties["key2"]); d != "" {
		t.Error("request key1 mismatch (-want +got):", d)
	}
	// error should not be set
	if _, ok := reqBody.Properties["error"]; ok {
		t.Error("request error is present")
	}
}

func TestSegmentClient_SuccessErr(t *testing.T) {
	httpErr := errors.New("http error")
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			return nil, httpErr
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()

	if err := cli.Success(ctx, Install); err == nil {
		t.Error("start call should have failed")
	} else if !errors.Is(err, httpErr) {
		t.Error("start call error should contain http error", err)
	}
}

func TestSegmentClient_Failure(t *testing.T) {
	var req *http.Request
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			req = r
			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()
	failure := errors.New("failure reason")

	if err := cli.Failure(ctx, Install, failure); err != nil {
		t.Error("start call failed", err)
	}

	// url
	if d := cmp.Diff(url, req.URL.String()); d != "" {
		t.Error("request URL mismatch (-want +got):", d)
	}
	// method
	if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
		t.Error("request method mismatch (-want +got):", d)
	}
	// content-type
	if d := cmp.Diff("application/json", req.Header.Get("Content-Type")); d != "" {
		t.Error("request header mismatch (-want +got):", d)
	}
	// body
	reqBodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Error("unable to read request body", err)
	}
	var reqBody body
	if err := json.Unmarshal(reqBodyRaw, &reqBody); err != nil {
		t.Error("unable to unmarshal request body", err)
	}

	if d := cmp.Diff(userID.String(), reqBody.ID); d != "" {
		t.Error("request ID mismatch (-want +got):", d)
	}

	if d := cmp.Diff(string(Install), reqBody.Event); d != "" {
		t.Error("request event mismatch (-want +got):", d)
	}

	if d := cmp.Diff(time.Now().UTC().Format(time.RFC3339), reqBody.Timestamp, cmpopts.EquateApproxTime(1*time.Second)); d != "" {
		t.Error("request timestamp mismatch (-want +got):", d)
	}

	if d := cmp.Diff(trackingKey, reqBody.WriteKey); d != "" {
		t.Error("request tracking key mismatch (-want +got):", d)
	}
	// body properties
	if d := cmp.Diff(10, len(reqBody.Properties)); d != "" {
		t.Error("request property count mismatch (-want +got):", d)
	}
	if d := cmp.Diff("abctl", reqBody.Properties["deployment_method"]); d != "" {
		t.Error("request deployment_method mismatch (-want +got):", d)
	}
	if d := cmp.Diff(sessionID.String(), reqBody.Properties["session_id"]); d != "" {
		t.Error("request session_id mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Failed), reqBody.Properties["state"]); d != "" {
		t.Error("request state mismatch (-want +got):", d)
	}
	if d := cmp.Diff(runtime.GOOS, reqBody.Properties["os"]); d != "" {
		t.Error("request os mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["build"]); d != "" {
		t.Error("request build mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["script_version"]); d != "" {
		t.Error("request script_version mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.Itoa(runtime.NumCPU()), reqBody.Properties["cpu_count"]); d != "" {
		t.Error("request cpu_count mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.FormatUint(memory.TotalMemory(), 10), reqBody.Properties["mem_total_bytes"]); d != "" {
		t.Error("request mem_total_bytes mismatch (-want +got):", d)
	}
	// free memory will fluctuate, only check it has a value greater than 0
	if v, ok := reqBody.Properties["mem_free_bytes"]; !ok {
		t.Error("request mem_free_bytes is missing")
	} else {
		free, _ := strconv.Atoi(v)
		if free <= 0 {
			t.Error("request mem_free_bytes is not set", v)
		}
	}
	// error should be set
	if d := cmp.Diff(failure.Error(), reqBody.Properties["error"]); d != "" {
		t.Error("request error mismatch (-want +got):", d)
	}
}

func TestSegmentClient_FailureWithAttr(t *testing.T) {
	var req *http.Request
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			req = r
			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)
	cli.Attr("key1", "val1")
	cli.Attr("key2", "val2")

	ctx := context.Background()
	failure := errors.New("failure reason")

	if err := cli.Failure(ctx, Install, failure); err != nil {
		t.Error("start call failed", err)
	}

	// url
	if d := cmp.Diff(url, req.URL.String()); d != "" {
		t.Error("request URL mismatch (-want +got):", d)
	}
	// method
	if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
		t.Error("request method mismatch (-want +got):", d)
	}
	// content-type
	if d := cmp.Diff("application/json", req.Header.Get("Content-Type")); d != "" {
		t.Error("request header mismatch (-want +got):", d)
	}
	// body
	reqBodyRaw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Error("unable to read request body", err)
	}
	var reqBody body
	if err := json.Unmarshal(reqBodyRaw, &reqBody); err != nil {
		t.Error("unable to unmarshal request body", err)
	}

	if d := cmp.Diff(userID.String(), reqBody.ID); d != "" {
		t.Error("request ID mismatch (-want +got):", d)
	}

	if d := cmp.Diff(string(Install), reqBody.Event); d != "" {
		t.Error("request event mismatch (-want +got):", d)
	}

	if d := cmp.Diff(time.Now().UTC().Format(time.RFC3339), reqBody.Timestamp, cmpopts.EquateApproxTime(1*time.Second)); d != "" {
		t.Error("request timestamp mismatch (-want +got):", d)
	}

	if d := cmp.Diff(trackingKey, reqBody.WriteKey); d != "" {
		t.Error("request tracking key mismatch (-want +got):", d)
	}
	// body properties
	if d := cmp.Diff(12, len(reqBody.Properties)); d != "" {
		t.Error("request property count mismatch (-want +got):", d)
	}
	if d := cmp.Diff("abctl", reqBody.Properties["deployment_method"]); d != "" {
		t.Error("request deployment_method mismatch (-want +got):", d)
	}
	if d := cmp.Diff(sessionID.String(), reqBody.Properties["session_id"]); d != "" {
		t.Error("request session_id mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Failed), reqBody.Properties["state"]); d != "" {
		t.Error("request state mismatch (-want +got):", d)
	}
	if d := cmp.Diff(runtime.GOOS, reqBody.Properties["os"]); d != "" {
		t.Error("request os mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["build"]); d != "" {
		t.Error("request build mismatch (-want +got):", d)
	}
	if d := cmp.Diff(build.Version, reqBody.Properties["script_version"]); d != "" {
		t.Error("request script_version mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.Itoa(runtime.NumCPU()), reqBody.Properties["cpu_count"]); d != "" {
		t.Error("request cpu_count mismatch (-want +got):", d)
	}
	if d := cmp.Diff(strconv.FormatUint(memory.TotalMemory(), 10), reqBody.Properties["mem_total_bytes"]); d != "" {
		t.Error("request mem_total_bytes mismatch (-want +got):", d)
	}
	// free memory will fluctuate, only check it has a value greater than 0
	if v, ok := reqBody.Properties["mem_free_bytes"]; !ok {
		t.Error("request mem_free_bytes is missing")
	} else {
		free, _ := strconv.Atoi(v)
		if free <= 0 {
			t.Error("request mem_free_bytes is not set", v)
		}
	}
	if d := cmp.Diff("val1", reqBody.Properties["key1"]); d != "" {
		t.Error("request key1 mismatch (-want +got):", d)
	}
	if d := cmp.Diff("val2", reqBody.Properties["key2"]); d != "" {
		t.Error("request key1 mismatch (-want +got):", d)
	}
	// error should be set
	if d := cmp.Diff(failure.Error(), reqBody.Properties["error"]); d != "" {
		t.Error("request error mismatch (-want +got):", d)
	}
}

func TestSegmentClient_FailureErr(t *testing.T) {
	httpErr := errors.New("http error")
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			return nil, httpErr
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()
	failure := errors.New("failure reason")

	if err := cli.Failure(ctx, Install, failure); err == nil {
		t.Error("start call should have failed")
	} else if !errors.Is(err, httpErr) {
		t.Error("start call error should contain http error", err)
	}
}

func TestSegmentClient_Wrap(t *testing.T) {
	var eventType *string
	var eventStates []string

	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal("failed to read body", err)
			}
			var body body
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatal("failed to unmarshal body", err)
			}

			if eventType == nil {
				eventType = &body.Event
			} else if d := cmp.Diff(*eventType, body.Event); d != "" {
				t.Errorf("event type mismatch (-want +got): %s", d)
			}
			eventStates = append(eventStates, body.Properties["state"])

			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()

	// a Wrap call where the func() error doesn't return an error
	// should call Start and Success
	if err := cli.Wrap(ctx, Install, func() error { return nil }); err != nil {
		t.Error("Wrap call failed")
	}

	if d := cmp.Diff(string(Install), *eventType); d != "" {
		t.Error("event type mismatch (-want +got):", d)
	}
	if d := cmp.Diff(2, len(eventStates)); d != "" {
		t.Fatal("eventStates size mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Start), eventStates[0]); d != "" {
		t.Error("start mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Success), eventStates[1]); d != "" {
		t.Error("success mismatch (-want +got):", d)
	}
}

func TestSegmentClient_WrapErr(t *testing.T) {
	var eventType *string
	var eventStates []string

	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal("failed to read body", err)
			}
			var body body
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatal("failed to unmarshal body", err)
			}

			if eventType == nil {
				eventType = &body.Event
			} else if d := cmp.Diff(*eventType, body.Event); d != "" {
				t.Errorf("event type mismatch (-want +got): %s", d)
			}
			eventStates = append(eventStates, body.Properties["state"])

			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}

	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}

	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)

	ctx := context.Background()

	actualErr := errors.New("test failure")

	// a Wrap call where the func() error returns asn error
	// should call Start and Failure
	err := cli.Wrap(ctx, Uninstall, func() error { return actualErr })
	if d := cmp.Diff(actualErr, err, cmpopts.EquateErrors()); d != "" {
		t.Errorf("error mismatch (-want +got): %s", err)
	}

	if d := cmp.Diff(string(Uninstall), *eventType); d != "" {
		t.Error("event type mismatch (-want +got):", d)
	}
	if d := cmp.Diff(2, len(eventStates)); d != "" {
		t.Fatal("eventStates size mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Start), eventStates[0]); d != "" {
		t.Error("start mismatch (-want +got):", d)
	}
	if d := cmp.Diff(string(Failed), eventStates[1]); d != "" {
		t.Error("failed mismatch (-want +got):", d)
	}
}

func TestSegmentClient_WrapPanic(t *testing.T) {
	mDoer := &mockDoer{
		do: func(r *http.Request) (*http.Response, error) {
			return &http.Response{Body: io.NopCloser(&strings.Reader{})}, nil
		},
	}
	opts := []Option{
		WithSessionID(sessionID),
		WithHTTPClient(mDoer),
	}
	ctx := context.Background()
	cli := NewSegmentClient(Config{AnalyticsID: UUID(userID)}, opts...)
	expect := errors.New("test panic err")

	err := cli.Wrap(ctx, Install, func() error {
		panic(expect)
	})
	if !errors.Is(err, expect) {
		t.Errorf("expected %q but got %q", expect, err)
	}

	expectStr := "panic: test str"
	err = cli.Wrap(ctx, Install, func() error {
		panic("test str")
	})
	if err == nil || err.Error() != expectStr {
		t.Errorf("expected %q but got %q", expectStr, err)
	}
}

// --- mocks
var _ Doer = (*mockDoer)(nil)

type mockDoer struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}
