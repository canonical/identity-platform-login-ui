// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sso "github.com/canonical/identity-platform-api/v0/sso"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

// Error reasons sso-service and tenant-service put in google.rpc.ErrorInfo.
const (
	reasonNotApplicable  = "NOT_APPLICABLE"
	reasonNotAdmitted    = "NOT_ADMITTED"
	reasonLastCredential = "LAST_CREDENTIAL"
	reasonHasTenant      = "HAS_TENANT"
)

var (
	// errNotApplicable is returned by StartAttempt for a connection that is
	// not tested, and by CompleteAttempt for an attempt that did not end at
	// the account, that the browser did not complete, or that has expired.
	errNotApplicable = errors.New("not applicable")
	// errLastCredential is returned when DeleteLink would leave the account
	// no way to sign in.
	errLastCredential = errors.New("cannot remove the only way to sign in")
	// errNoLink is returned when the account has no link to the connection.
	errNoLink = errors.New("no such link")
)

// Option is one company sign-in offered for a tenant.
type Option struct {
	ConnectionID string
	Label        string
}

// AttemptRequest is the request of StartAttempt.
type AttemptRequest struct {
	TenantID       string
	Email          string
	ConnectionID   string
	Reauthenticate bool
}

// Completion is the answer of CompleteAttempt.
type Completion struct {
	ConnectionID string
	TenantID     string
}

// Link is one company sign-in linked to an account.
type Link struct {
	ConnectionID string
	Label        string
	TenantID     string
}

// ssoServiceRetried are the sso-service calls retried on UNAVAILABLE:
// repeated, each leaves what it left the first time. DeleteLink is not one
// of them: a second call, after the answer of the first one was lost, would
// report a link that was removed as one that never existed.
var ssoServiceRetried = []string{
	sso.SSOSignInService_ListOptions_FullMethodName,
	sso.SSOSignInService_StartAttempt_FullMethodName,
	sso.SSOSignInService_CompleteAttempt_FullMethodName,
	sso.SSOSignInService_ListLinks_FullMethodName,
}

// SSOServiceDialOptions returns the dial options of the channel to
// sso-service.
func SSOServiceDialOptions(tokens oauth2.TokenSource) []grpc.DialOption {
	return dialOptions(tokens, ssoServiceRetried)
}

// SSO calls the sign-in API of sso-service.
type SSO struct {
	client  SSOServiceClientInterface
	timeout time.Duration

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (s *SSO) Options(ctx context.Context, connectionIDs []string) ([]Option, error) {
	if len(connectionIDs) == 0 {
		return nil, nil
	}

	ctx, span := s.tracer.Start(ctx, "byosso.SSO.Options")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.ListOptions(ctx, &sso.ListOptionsRequest{ConnectionIds: connectionIDs})
	if err != nil {
		s.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot list company sign-ins")
		return nil, fmt.Errorf("cannot list company sign-ins: %w", err)
	}

	options := make([]Option, 0, len(resp.GetOptions()))
	for _, o := range resp.GetOptions() {
		options = append(options, Option{ConnectionID: o.GetConnectionId(), Label: o.GetLabel()})
	}

	span.SetStatus(codes.Ok, "")
	return options, nil
}

func (s *SSO) StartAttempt(ctx context.Context, req AttemptRequest) (string, error) {
	ctx, span := s.tracer.Start(ctx, "byosso.SSO.StartAttempt")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.StartAttempt(ctx, &sso.StartAttemptRequest{
		TenantId:       req.TenantID,
		Email:          req.Email,
		ConnectionId:   req.ConnectionID,
		Reauthenticate: req.Reauthenticate,
	})
	if err != nil {
		s.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot start a company sign-in")
		return "", fmt.Errorf("cannot start a company sign-in: %w", ssoError(err))
	}

	if resp.GetTicket() == "" {
		err := fmt.Errorf("empty ticket from sso-service")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	return resp.GetTicket(), nil
}

func (s *SSO) CompleteAttempt(ctx context.Context, ticket, identityID, receipt string) (*Completion, error) {
	ctx, span := s.tracer.Start(ctx, "byosso.SSO.CompleteAttempt")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.CompleteAttempt(ctx, &sso.CompleteAttemptRequest{Ticket: ticket, IdentityId: identityID, Receipt: receipt})
	if err != nil {
		s.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot complete the company sign-in")
		return nil, fmt.Errorf("cannot complete the company sign-in: %w", ssoError(err))
	}

	span.SetStatus(codes.Ok, "")
	return &Completion{ConnectionID: resp.GetConnectionId(), TenantID: resp.GetTenantId()}, nil
}

func (s *SSO) Links(ctx context.Context, identityID string) ([]Link, error) {
	ctx, span := s.tracer.Start(ctx, "byosso.SSO.Links")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.client.ListLinks(ctx, &sso.ListLinksRequest{IdentityId: identityID})
	if err != nil {
		s.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot list the company sign-ins of an identity")
		return nil, fmt.Errorf("cannot list the company sign-ins of %s: %w", identityID, err)
	}

	links := make([]Link, 0, len(resp.GetLinks()))
	for _, l := range resp.GetLinks() {
		links = append(links, Link{ConnectionID: l.GetConnectionId(), Label: l.GetLabel(), TenantID: l.GetTenantId()})
	}

	span.SetStatus(codes.Ok, "")
	return links, nil
}

func (s *SSO) DeleteLink(ctx context.Context, connectionID, identityID string) error {
	ctx, span := s.tracer.Start(ctx, "byosso.SSO.DeleteLink")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if _, err := s.client.DeleteLink(ctx, &sso.DeleteLinkRequest{ConnectionId: connectionID, IdentityId: identityID}); err != nil {
		s.logger.Debugf("full gRPC status: %v", status.Convert(err).Proto())
		span.RecordError(err)
		span.SetStatus(codes.Error, "cannot remove the company sign-in")
		return fmt.Errorf("cannot remove the company sign-in: %w", ssoError(err))
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ssoError turns the ErrorInfo reasons of sso-service into sentinel errors.
func ssoError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch errorReason(st) {
	case reasonNotApplicable:
		return errNotApplicable
	case reasonLastCredential:
		return errLastCredential
	}
	if st.Code() == grpccodes.NotFound {
		return errNoLink
	}
	return err
}

// errorReason returns the ErrorInfo reason of a gRPC status, or "".
func errorReason(st *status.Status) string {
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok {
			return info.GetReason()
		}
	}
	return ""
}

func NewSSO(client SSOServiceClientInterface, timeout time.Duration, tracer tracing.TracingInterface, logger logging.LoggerInterface) *SSO {
	s := new(SSO)

	if timeout <= 0 {
		timeout = defaultGRPCTimeout
	}

	s.client = client
	s.timeout = timeout

	s.tracer = tracer
	s.logger = logger

	return s
}
