// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kClient "github.com/ory/kratos-client-go/v25"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// underBudget matches the context of a call made within the request budget.
type underBudget struct{}

func (underBudget) Matches(x any) bool {
	ctx, ok := x.(context.Context)
	if !ok {
		return false
	}
	deadline, ok := ctx.Deadline()
	return ok && time.Until(deadline) <= requestBudget
}

func (underBudget) String() string {
	return "is a context under the request budget"
}

// statusWithReason builds a gRPC error carrying an ErrorInfo reason.
func statusWithReason(code codes.Code, reason string) error {
	st, _ := status.New(code, "refused").WithDetails(&errdetails.ErrorInfo{Reason: reason})
	return st.Err()
}

func TestIsUnavailable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "no error", err: nil, expected: false},
		{name: "unavailable", err: status.Error(codes.Unavailable, "down"), expected: true},
		{name: "unavailable and wrapped", err: tenantServiceUnavailable(), expected: true},
		{name: "deadline exceeded", err: status.Error(codes.DeadlineExceeded, "slow"), expected: true},
		{name: "request budget ran out", err: fmt.Errorf("kratos: %w", context.DeadlineExceeded), expected: true},
		{name: "refusal with a reason", err: statusWithReason(codes.FailedPrecondition, "NOT_ADMITTED"), expected: false},
		{name: "not found", err: status.Error(codes.NotFound, "no such tenant"), expected: false},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "bad token"), expected: false},
		{name: "internal", err: status.Error(codes.Internal, "error"), expected: false},
		{name: "not a gRPC error", err: errors.New("error"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if unavailable := isUnavailable(tt.err); unavailable != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, unavailable)
			}
		})
	}
}

func TestRequestFields(t *testing.T) {
	tests := []struct {
		name        string
		request     func() *http.Request
		expected    string
		expectedRaw string
	}{
		{
			name: "json body",
			request: func() *http.Request {
				return jsonRequest(http.MethodPost, "/", `{"sso_connection":"test"}`)
			},
			expected:    "test",
			expectedRaw: `{"sso_connection":"test"}`,
		},
		{
			name: "form body",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("sso_connection=test"))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			},
			expected:    "test",
			expectedRaw: "sso_connection=test",
		},
		{
			name: "query",
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/?sso_connection=test", nil)
			},
			expected:    "test",
			expectedRaw: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.request()

			fields, err := requestFields(r)

			expectNoError(t, err)
			if value := stringField(fields, "sso_connection"); value != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, value)
			}
			// the body is left for the next reader
			raw, err := peekRequestBody(r)
			expectNoError(t, err)
			if string(raw) != tt.expectedRaw {
				t.Fatalf("expected body %q, got %q", tt.expectedRaw, raw)
			}
		})
	}
}

func TestAddressUnverified(t *testing.T) {
	withAddress := func(verified bool) *kClient.Session {
		session := companySession("s")
		session.Identity.VerifiableAddresses = []kClient.VerifiableIdentityAddress{
			*kClient.NewVerifiableIdentityAddress("pending", testEmail, verified, "email"),
		}
		return session
	}

	tests := []struct {
		name     string
		session  *kClient.Session
		expected bool
	}{
		{name: "unverified address", session: withAddress(false), expected: true},
		{name: "verified address", session: withAddress(true), expected: false},
		{name: "no verifiable address", session: companySession("s"), expected: false},
		{name: "no session", session: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if unverified := addressUnverified(tt.session); unverified != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, unverified)
			}
		})
	}
}
