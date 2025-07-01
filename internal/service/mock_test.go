package service

import (
	"net/http"
)

var _ HTTPClient = (*mockHTTP)(nil)

type mockHTTP struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTP) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}
