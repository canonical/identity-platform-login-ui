// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sso "github.com/canonical/identity-platform-api/v0/sso"
)

func TestSSOSuccess(t *testing.T) {
	ctx := context.Background()

	// each call is made once, with a deadline, and its answer is returned
	tests := []struct {
		name     string
		span     string
		call     func(mockClient *MockSSOServiceClientInterface, s *SSO) (any, error)
		expected any
	}{
		{name: "Options", span: "byosso.SSO.Options", expected: []Option{{ConnectionID: connA, Label: "Acme"}}, call: func(mockClient *MockSSOServiceClientInterface, s *SSO) (any, error) {
			mockClient.EXPECT().ListOptions(underBudget{}, &sso.ListOptionsRequest{ConnectionIds: []string{connA, connB}}).Times(1).Return(
				&sso.ListOptionsResponse{Options: []*sso.Option{{ConnectionId: connA, Label: "Acme"}}}, nil,
			)
			return s.Options(ctx, []string{connA, connB})
		}},
		{name: "StartAttempt", span: "byosso.SSO.StartAttempt", expected: "ticket", call: func(mockClient *MockSSOServiceClientInterface, s *SSO) (any, error) {
			mockClient.EXPECT().StartAttempt(underBudget{}, &sso.StartAttemptRequest{TenantId: testTenant, Email: testEmail, ConnectionId: connA, Reauthenticate: true}).Times(1).Return(
				&sso.StartAttemptResponse{Ticket: "ticket"}, nil,
			)
			return s.StartAttempt(ctx, AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA, Reauthenticate: true})
		}},
		{name: "CompleteAttempt", span: "byosso.SSO.CompleteAttempt", expected: &Completion{ConnectionID: connA, TenantID: testTenant}, call: func(mockClient *MockSSOServiceClientInterface, s *SSO) (any, error) {
			mockClient.EXPECT().CompleteAttempt(underBudget{}, &sso.CompleteAttemptRequest{Ticket: "ticket", IdentityId: "iid", Receipt: "receipt"}).Times(1).Return(
				&sso.CompleteAttemptResponse{ConnectionId: connA, TenantId: testTenant}, nil,
			)
			return s.CompleteAttempt(ctx, "ticket", "iid", "receipt")
		}},
		{name: "Links", span: "byosso.SSO.Links", expected: []Link{{ConnectionID: connA, Label: "Acme", TenantID: testTenant}}, call: func(mockClient *MockSSOServiceClientInterface, s *SSO) (any, error) {
			mockClient.EXPECT().ListLinks(underBudget{}, &sso.ListLinksRequest{IdentityId: "iid"}).Times(1).Return(
				&sso.ListLinksResponse{Links: []*sso.Link{{ConnectionId: connA, Label: "Acme", TenantId: testTenant}}}, nil,
			)
			return s.Links(ctx, "iid")
		}},
		{name: "DeleteLink", span: "byosso.SSO.DeleteLink", call: func(mockClient *MockSSOServiceClientInterface, s *SSO) (any, error) {
			mockClient.EXPECT().DeleteLink(underBudget{}, &sso.DeleteLinkRequest{ConnectionId: connA, IdentityId: "iid"}).Times(1).Return(&sso.DeleteLinkResponse{}, nil)
			return nil, s.DeleteLink(ctx, connA, "iid")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := NewMockLoggerInterface(ctrl)
			mockClient := NewMockSSOServiceClientInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)

			mockTracer.EXPECT().Start(ctx, tt.span).Times(1).Return(ctx, trace.SpanFromContext(ctx))

			got, err := tt.call(mockClient, NewSSO(mockClient, time.Second, mockTracer, mockLogger))

			expectNoError(t, err)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("expected %+v, got %+v", tt.expected, got)
			}
		})
	}
}

func TestSSOFails(t *testing.T) {
	ctx := context.Background()

	// expected is the refusal the error is, nil when it is none of them
	tests := []struct {
		name                string
		call                func(mockClient *MockSSOServiceClientInterface, s *SSO) error
		expected            error
		expectedUnavailable bool
	}{
		{name: "attempt of another account", expected: errNotApplicable, call: func(mockClient *MockSSOServiceClientInterface, s *SSO) error {
			mockClient.EXPECT().CompleteAttempt(gomock.Any(), gomock.Any()).Times(1).Return(nil, statusWithReason(codes.FailedPrecondition, "NOT_APPLICABLE"))
			_, err := s.CompleteAttempt(ctx, "ticket", "iid", "")
			return err
		}},
		{name: "last credential", expected: errLastCredential, call: func(mockClient *MockSSOServiceClientInterface, s *SSO) error {
			mockClient.EXPECT().DeleteLink(gomock.Any(), gomock.Any()).Times(1).Return(nil, statusWithReason(codes.FailedPrecondition, "LAST_CREDENTIAL"))
			return s.DeleteLink(ctx, connA, "iid")
		}},
		{name: "no link", expected: errNoLink, call: func(mockClient *MockSSOServiceClientInterface, s *SSO) error {
			mockClient.EXPECT().DeleteLink(gomock.Any(), gomock.Any()).Times(1).Return(nil, status.Error(codes.NotFound, "no link"))
			return s.DeleteLink(ctx, connA, "iid")
		}},
		// an outage is not a refusal
		{name: "unavailable", expectedUnavailable: true, call: func(mockClient *MockSSOServiceClientInterface, s *SSO) error {
			mockClient.EXPECT().StartAttempt(gomock.Any(), gomock.Any()).Times(1).Return(nil, status.Error(codes.Unavailable, "down"))
			_, err := s.StartAttempt(ctx, AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
			return err
		}},
		{name: "empty ticket", call: func(mockClient *MockSSOServiceClientInterface, s *SSO) error {
			mockClient.EXPECT().StartAttempt(gomock.Any(), gomock.Any()).Times(1).Return(&sso.StartAttemptResponse{}, nil)
			_, err := s.StartAttempt(ctx, AttemptRequest{TenantID: testTenant, Email: testEmail, ConnectionID: connA})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := NewMockLoggerInterface(ctrl)
			mockClient := NewMockSSOServiceClientInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)

			mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
			mockTracer.EXPECT().Start(ctx, gomock.Any()).Times(1).Return(ctx, trace.SpanFromContext(ctx))

			err := tt.call(mockClient, NewSSO(mockClient, time.Second, mockTracer, mockLogger))

			if err == nil {
				t.Fatalf("expected error not nil")
			}
			for _, refusal := range []error{errNotApplicable, errLastCredential, errNoLink} {
				if is := errors.Is(err, refusal); is != (refusal == tt.expected) {
					t.Fatalf("expected refusal %q to be %v, got %v", refusal, refusal == tt.expected, is)
				}
			}
			if unavailable := isUnavailable(err); unavailable != tt.expectedUnavailable {
				t.Fatalf("expected unavailable %v, got %v", tt.expectedUnavailable, unavailable)
			}
		})
	}
}
