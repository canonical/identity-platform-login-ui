// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	sso "github.com/canonical/identity-platform-api/v0/sso"
	tenant "github.com/canonical/identity-platform-api/v0/tenant"

	ig "github.com/canonical/identity-platform-login-ui/internal/grpc"
)

var testTokens = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"})

// failingTokens is a token source that has no token to give.
type failingTokens struct {
	err error
}

func (f failingTokens) Token() (*oauth2.Token, error) {
	return nil, f.err
}

// recordingServer answers every method of every service: the first failures calls
// of a method with code, the rest with an empty message. It records how
// often each method was called, whether a call came with a deadline, and the
// authorization it carried.
type recordingServer struct {
	code     codes.Code
	failures int

	mu            sync.Mutex
	calls         map[string]int
	deadlines     map[string]bool
	authorization map[string]string
}

func (s *recordingServer) handle(_ any, stream grpc.ServerStream) error {
	method, _ := grpc.MethodFromServerStream(stream)
	s.mu.Lock()
	s.calls[method]++
	n := s.calls[method]
	if _, ok := stream.Context().Deadline(); ok {
		s.deadlines[method] = true
	}
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok && len(md.Get("authorization")) > 0 {
		s.authorization[method] = md.Get("authorization")[0]
	}
	s.mu.Unlock()

	if n <= s.failures {
		return status.Error(s.code, "not now")
	}
	if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
		return err
	}
	return stream.SendMsg(&emptypb.Empty{})
}

func (s *recordingServer) called(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[method]
}

// dialTestServer starts a recordingServer and dials it with opts.
func dialTestServer(t *testing.T, code codes.Code, failures int, opts ...grpc.DialOption) (*recordingServer, *grpc.ClientConn) {
	t.Helper()
	recording := &recordingServer{code: code, failures: failures, calls: map[string]int{}, deadlines: map[string]bool{}, authorization: map[string]string{}}
	listener := bufconn.Listen(1 << 16)
	server := grpc.NewServer(grpc.UnknownServiceHandler(recording.handle))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return listener.DialContext(ctx)
	}))
	conn, err := ig.NewConn("test", "passthrough:///test", false, opts...)
	expectNoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return recording, conn
}

// invoke calls method with an empty message.
func invoke(conn *grpc.ClientConn, method string) error {
	return conn.Invoke(context.Background(), method, &emptypb.Empty{}, &emptypb.Empty{})
}

func TestDialOptionsRetry(t *testing.T) {
	tenantService, ssoService := TenantServiceDialOptions(testTokens), SSOServiceDialOptions(testTokens)

	// the service answers the first failures calls of a method with code
	tests := []struct {
		name          string
		opts          []grpc.DialOption
		methods       []string
		code          codes.Code
		failures      int
		expectedCode  codes.Code
		expectedCalls int
	}{
		// the calls that are safe to repeat succeed at the third attempt
		{name: "tenant-service unavailable twice", opts: tenantService, methods: tenantServiceRetried, code: codes.Unavailable, failures: 2, expectedCode: codes.OK, expectedCalls: 3},
		{name: "sso-service unavailable twice", opts: ssoService, methods: ssoServiceRetried, code: codes.Unavailable, failures: 2, expectedCode: codes.OK, expectedCalls: 3},
		{name: "unavailable every time", opts: tenantService, methods: tenantServiceRetried[:1], code: codes.Unavailable, failures: 100, expectedCode: codes.Unavailable, expectedCalls: 3},
		{name: "answer of the service", opts: tenantService, methods: []string{tenant.TenantSignInService_JoinTenant_FullMethodName}, code: codes.FailedPrecondition, failures: 100, expectedCode: codes.FailedPrecondition, expectedCalls: 1},
		{name: "write that is not safe to repeat", opts: tenantService, methods: []string{tenant.TenantService_InviteMember_FullMethodName}, code: codes.Unavailable, failures: 100, expectedCode: codes.Unavailable, expectedCalls: 1},
		// a removal whose answer was lost may have happened: it is not asked for again
		{name: "removal of a link", opts: ssoService, methods: []string{sso.SSOSignInService_DeleteLink_FullMethodName}, code: codes.Unavailable, failures: 1, expectedCode: codes.Unavailable, expectedCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recording, conn := dialTestServer(t, tt.code, tt.failures, tt.opts...)

			for _, method := range tt.methods {
				err := invoke(conn, method)

				if code := status.Code(err); code != tt.expectedCode {
					t.Fatalf("expected code %s for %s, got %v", tt.expectedCode, method, err)
				}
				if calls := recording.called(method); calls != tt.expectedCalls {
					t.Fatalf("expected %d calls of %s, got %d", tt.expectedCalls, method, calls)
				}
				if authorization := recording.authorization[method]; authorization != "Bearer tok" {
					t.Fatalf("expected authorization Bearer tok for %s, got %s", method, authorization)
				}
				// the deadline of a call is the one its caller sets
				if recording.deadlines[method] {
					t.Fatalf("expected no deadline for %s", method)
				}
			}
		})
	}
}

func TestDialOptionsFailOnToken(t *testing.T) {
	// a token endpoint that cannot answer is an outage, one that refuses the client is not
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{name: "token endpoint not reached", err: errors.New("dial tcp: connection refused"), expectedCode: codes.Unavailable},
		{name: "server error", err: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusBadGateway}}, expectedCode: codes.Unavailable},
		{name: "client refused", err: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusUnauthorized}}, expectedCode: codes.Unauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recording, conn := dialTestServer(t, codes.OK, 0, TenantServiceDialOptions(failingTokens{err: tt.err})...)

			err := invoke(conn, tenant.TenantService_LookupTenants_FullMethodName)

			if code := status.Code(err); code != tt.expectedCode {
				t.Fatalf("expected code %s, got %v", tt.expectedCode, err)
			}
			// the call never reaches the service without a token
			if calls := recording.called(tenant.TenantService_LookupTenants_FullMethodName); calls != 0 {
				t.Fatalf("expected no call, got %d", calls)
			}
		})
	}
}

func TestNewServiceTokenSource(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("expected error to be nil, got %v", err)
		}
		if grantType := r.Form.Get("grant_type"); grantType != "client_credentials" {
			t.Errorf("expected grant type client_credentials, got %s", grantType)
		}
		if scope := r.Form.Get("scope"); scope != "tenant-service" {
			t.Errorf("expected scope tenant-service, got %s", scope)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	token, err := NewServiceTokenSource(tokenServer.URL, "login-ui", "secret", []string{"tenant-service"}).Token()

	expectNoError(t, err)
	if token.AccessToken != "tok" {
		t.Fatalf("expected access token tok, got %s", token.AccessToken)
	}
}

func TestNewTokenSourceTimeout(t *testing.T) {
	release := make(chan struct{})
	tokenServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer tokenServer.Close()
	defer close(release)

	start := time.Now()
	tokens := newTokenSource(50*time.Millisecond, tokenServer.URL, "login-ui", "secret", nil)
	_, err := bearerCredentials{tokens: tokens}.GetRequestMetadata(context.Background())

	// a token endpoint that does not answer is given up on, as an outage
	if !isUnavailable(err) {
		t.Fatalf("expected an unavailable error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("expected the token request to time out, took %v", elapsed)
	}
}
