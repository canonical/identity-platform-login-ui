// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	httpHelpers "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

// Verification starts the verification of an address through the public API
// of Kratos.
type Verification struct {
	kratos KratosClientInterface

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

// Start creates a verification flow that returns to returnTo and submits
// email to it, so that the browser can only enter the code sent to that
// address. It returns the ID of the flow and the cookies of Kratos.
func (v *Verification) Start(ctx context.Context, returnTo, email string, cookies []*http.Cookie) (string, []*http.Cookie, error) {
	ctx, span := v.tracer.Start(ctx, "byosso.Verification.Start")
	defer span.End()

	flow, res, err := v.kratos.FrontendApi().CreateBrowserVerificationFlow(ctx).ReturnTo(returnTo).Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		v.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", nil, fmt.Errorf("unable to create verification flow: %w", err)
	}

	var flowCookies []*http.Cookie
	if res != nil {
		flowCookies = res.Cookies()
	}

	body := kClient.NewUpdateVerificationFlowWithCodeMethod(methodCode)
	body.SetEmail(email)
	if csrfToken := flowInputValue(flow.Ui.Nodes, "csrf_token"); csrfToken != "" {
		body.SetCsrfToken(csrfToken)
	}

	_, res, err = v.kratos.FrontendApi().UpdateVerificationFlow(ctx).
		Flow(flow.Id).
		Cookie(httpHelpers.CookiesToString(mergeCookies(cookies, flowCookies))).
		UpdateVerificationFlowBody(kClient.UpdateVerificationFlowWithCodeMethodAsUpdateVerificationFlowBody(body)).
		Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		v.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", nil, fmt.Errorf("unable to update verification flow: %w", err)
	}

	if res != nil {
		flowCookies = mergeCookies(flowCookies, res.Cookies())
	}

	span.SetStatus(codes.Ok, "")
	return flow.Id, flowCookies, nil
}

// IdentityFinder looks identities up through the admin API of Kratos.
type IdentityFinder struct {
	kratosAdmin KratosAdminClientInterface

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (f *IdentityFinder) IdentityExists(ctx context.Context, email string) (bool, error) {
	id, err := f.IdentityID(ctx, email)
	return id != "", err
}

// IdentityID returns the ID of the identity holding email among its
// credential identifiers, "" when there is none.
func (f *IdentityFinder) IdentityID(ctx context.Context, email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", nil
	}

	ctx, span := f.tracer.Start(ctx, "kratos.IdentityAPI.ListIdentities")
	defer span.End()

	identities, res, err := f.kratosAdmin.IdentityApi().ListIdentities(ctx).CredentialsIdentifier(email).Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		f.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	if len(identities) == 0 {
		return "", nil
	}
	return identities[0].GetId(), nil
}

// HasRecoveryCodes reports whether the identity holds recovery codes.
func (f *IdentityFinder) HasRecoveryCodes(ctx context.Context, identityID string) (bool, error) {
	ctx, span := f.tracer.Start(ctx, "kratos.IdentityAPI.GetIdentity")
	defer span.End()

	identity, res, err := f.kratosAdmin.IdentityApi().GetIdentity(ctx, identityID).IncludeCredential([]string{methodLookup}).Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		f.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	span.SetStatus(codes.Ok, "")
	_, ok := identity.GetCredentials()[methodLookup]
	return ok, nil
}

func NewVerification(kratos KratosClientInterface, tracer tracing.TracingInterface, logger logging.LoggerInterface) *Verification {
	v := new(Verification)

	v.kratos = kratos

	v.tracer = tracer
	v.logger = logger

	return v
}

func NewIdentityFinder(kratosAdmin KratosAdminClientInterface, tracer tracing.TracingInterface, logger logging.LoggerInterface) *IdentityFinder {
	f := new(IdentityFinder)

	f.kratosAdmin = kratosAdmin

	f.tracer = tracer
	f.logger = logger

	return f
}
