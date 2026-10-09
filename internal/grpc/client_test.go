// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package grpc

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNewConn(t *testing.T) {
	var calls atomic.Int32
	listener := bufconn.Listen(1 << 16)
	server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, _ grpc.ServerStream) error {
		calls.Add(1)
		return status.Error(codes.Unavailable, "not now")
	}))
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()

	conn, err := NewConn("test", "passthrough:///test", false, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return listener.DialContext(ctx)
	}))

	if err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
	defer conn.Close()

	err = conn.Invoke(context.Background(), "/test.Service/Method", &emptypb.Empty{}, &emptypb.Empty{})

	if code := status.Code(err); code != codes.Unavailable {
		t.Fatalf("expected code %s, got %s", codes.Unavailable, code)
	}
	// a call is made once unless the dial options say otherwise
	if n := calls.Load(); n != 1 {
		t.Fatalf("expected 1 call, got %d", n)
	}
}

func TestNewConnWithTLS(t *testing.T) {
	conn, err := NewConn("test", "passthrough:///test", true)

	if err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
	defer conn.Close()
}

func TestNewConnFailOnDialOption(t *testing.T) {
	conn, err := NewConn("test", "passthrough:///test", false, grpc.WithDefaultServiceConfig("{"))

	if conn != nil {
		t.Fatalf("expected connection to be nil, got %v", conn)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}
