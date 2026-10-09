// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package grpc

import (
	"crypto/tls"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	keepaliveTime    = 30 * time.Second
	keepaliveTimeout = 10 * time.Second
)

// NewConn creates a gRPC client connection to the named service at address,
// configured with keepalive parameters and optionally TLS transport
// credentials. The caller is responsible for closing the returned connection.
func NewConn(service, address string, tlsEnabled bool, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	var creds credentials.TransportCredentials
	if tlsEnabled {
		creds = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	} else {
		creds = insecure.NewCredentials()
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                keepaliveTime,
			Timeout:             keepaliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	conn, err := grpc.NewClient(address, append(dialOpts, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s at %s: %v", service, address, err)
	}

	return conn, nil
}
