// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

// LoginRequest is what the accept checks read from a Hydra login request.
type LoginRequest struct {
	PromptLogin bool
	// MaxAge is nil when the app sent none.
	MaxAge *time.Duration
	// Subject is the subject Hydra remembers in the login session of this
	// browser, SessionID that session and RequestURL the authorization
	// request of the app.
	Subject    string
	SessionID  string
	RequestURL string
}

// RemembersAnother reports whether Hydra remembers a subject other than
// identityID in this browser. Accepting identityID then makes Hydra start
// the request of the app again with prompt=login added, which would be read
// as the app asking for a fresh sign-in the user has just given.
func (lr *LoginRequest) RemembersAnother(identityID string) bool {
	return lr != nil && lr.Subject != "" && lr.Subject != identityID && lr.SessionID != "" && lr.RequestURL != ""
}

// Reauthenticate reports whether the company sign-in must ask the IdP for a
// fresh login: prompt=login, or a max_age shorter than the time since the
// first factor of the session. Without a session the first factor is fresh
// anyway, and only prompt=login or max_age=0 ask for it.
func (lr *LoginRequest) Reauthenticate(session *kClient.Session, now time.Time) bool {
	if lr == nil {
		return false
	}
	if lr.PromptLogin {
		return true
	}
	if lr.MaxAge == nil {
		return false
	}
	if session == nil {
		return *lr.MaxAge == 0
	}
	return lr.NeedsFreshFirstFactor(session, now)
}

// NeedsFreshFirstFactor reports whether the first factor of the session is
// too old for this request: prompt=login, or a max_age shorter than the time
// since the first factor. It is judged on the Kratos session, not on the skip
// of Hydra.
func (lr *LoginRequest) NeedsFreshFirstFactor(session *kClient.Session, now time.Time) bool {
	if lr == nil {
		return false
	}
	if lr.PromptLogin {
		return true
	}
	if lr.MaxAge == nil {
		return false
	}
	f := firstMethod(session)
	if f == nil || f.CompletedAt == nil {
		return true
	}
	return now.Sub(*f.CompletedAt) > *lr.MaxAge
}

// parseLoginRequestURL reads prompt and max_age from an authorization URL.
func parseLoginRequestURL(requestURL string) *LoginRequest {
	lr := &LoginRequest{}
	u, err := url.Parse(requestURL)
	if err != nil {
		return lr
	}
	q := u.Query()
	for _, p := range strings.Fields(q.Get("prompt")) {
		if p == "login" {
			lr.PromptLogin = true
		}
	}
	if v := q.Get("max_age"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			d := time.Duration(n) * time.Second
			lr.MaxAge = &d
		}
	}
	return lr
}

// Hydra reads the login requests of Hydra, ends its login sessions and
// rejects its consent requests.
type Hydra struct {
	hydra HydraClientInterface

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (h *Hydra) LoginRequest(ctx context.Context, loginChallenge string) (*LoginRequest, error) {
	ctx, span := h.tracer.Start(ctx, "hydra.OAuth2API.GetOAuth2LoginRequest")
	defer span.End()

	req, res, err := h.hydra.OAuth2API().GetOAuth2LoginRequest(ctx).LoginChallenge(loginChallenge).Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		h.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	lr := parseLoginRequestURL(req.GetRequestUrl())
	lr.Subject = req.GetSubject()
	lr.SessionID = req.GetSessionId()
	lr.RequestURL = req.GetRequestUrl()

	span.SetStatus(codes.Ok, "")
	return lr, nil
}

// RevokeLoginSession ends one login session by its sid: that session only,
// with OpenID Connect back-channel logout.
func (h *Hydra) RevokeLoginSession(ctx context.Context, sid string) error {
	ctx, span := h.tracer.Start(ctx, "hydra.OAuth2API.RevokeOAuth2LoginSessions")
	defer span.End()

	res, err := h.hydra.OAuth2API().RevokeOAuth2LoginSessions(ctx).Sid(sid).Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		h.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// RejectConsent rejects the consent request with login_required. It returns
// where the browser goes.
func (h *Hydra) RejectConsent(ctx context.Context, consentChallenge, description string) (string, error) {
	ctx, span := h.tracer.Start(ctx, "hydra.OAuth2API.RejectOAuth2ConsentRequest")
	defer span.End()

	redirectTo, res, err := h.hydra.OAuth2API().RejectOAuth2ConsentRequest(ctx).
		ConsentChallenge(consentChallenge).
		RejectOAuth2Request(loginRequired(description)).
		Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		h.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	return redirectTo.GetRedirectTo(), nil
}

func loginRequired(description string) hClient.RejectOAuth2Request {
	reject := hClient.NewRejectOAuth2Request()
	reject.SetError("login_required")
	reject.SetErrorDescription(description)
	return *reject
}

func NewHydra(hydra HydraClientInterface, tracer tracing.TracingInterface, logger logging.LoggerInterface) *Hydra {
	h := new(Hydra)

	h.hydra = hydra

	h.tracer = tracer
	h.logger = logger

	return h
}
