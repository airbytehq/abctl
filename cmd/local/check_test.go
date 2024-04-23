package local

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestPortAvailable_Available(t *testing.T) {
	// spin up a listener to find a port and then shut it down to ensure that port is available
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("could not create listener", err)
	}
	p := port(listener.Addr().String())
	if err := listener.Close(); err != nil {
		t.Fatal("could not close listener", err)
	}

	avail, err := portAvailable(context.Background(), p)
	if d := cmp.Diff(true, avail); d != "" {
		t.Error("portAvailable returned unexpected results", d)
	}
	if err != nil {
		t.Error("portAvailable returned unexpected error", err)
	}
}

func TestPortAvailable_Unavailable(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("could not create listener", err)
	}
	defer listener.Close()
	p := port(listener.Addr().String())

	avail, err := portAvailable(context.Background(), p)
	if d := cmp.Diff(false, avail); d != "" {
		t.Error("portAvailable returned unexpected results", d)
	}
	// expecting an error
	if err == nil {
		t.Error("portAvailable returned nil error")
	}
}

// port returns the port from a string value
func port(s string) int {
	vals := strings.Split(s, ":")
	p, _ := strconv.Atoi(vals[len(vals)-1])
	return p
}
