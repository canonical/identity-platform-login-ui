// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package extra

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
)

//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_extra.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_kratos.go github.com/ory/kratos-client-go/v25 FrontendAPI
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_hydra.go -source=../../internal/hydra/interfaces.go

func TestGetConsentSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	challengeString := "test.challenge"
	consentRequest := hClient.OAuth2APIGetOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	consent := hClient.NewOAuth2ConsentRequest(challengeString)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.GetOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequest(ctx).Times(1).Return(consentRequest)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIGetOAuth2ConsentRequestRequest) (*hClient.OAuth2ConsentRequest, *http.Response, error) {
			// use reflect as consentChallenge is a private attribute, also is a string pointer so need to cast it multiple times
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != challengeString {
				t.Fatalf("expected challenge string as %s, got %s", challengeString, *challenge)
			}

			return consent, new(http.Response), nil
		},
	)

	c, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).GetConsent(ctx, challengeString)

	if c != consent {
		t.Fatalf("expected consent to be %v not  %v", consent, c)
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}
func TestGetConsentFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	challengeString := "test.challenge"
	consentRequest := hClient.OAuth2APIGetOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.GetOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequest(ctx).Times(1).Return(consentRequest)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIGetOAuth2ConsentRequestRequest) (*hClient.OAuth2ConsentRequest, *http.Response, error) {
			// use reflect as consentChallenge is a private attribute, also is a string pointer so need to cast it multiple times
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != challengeString {
				t.Fatalf("expected challenge string as %s, got %s", challengeString, *challenge)
			}

			return nil, new(http.Response), fmt.Errorf("error")
		},
	)

	c, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).GetConsent(ctx, challengeString)

	if c != nil {
		t.Fatalf("expected consent to be nil not  %v", c)
	}

	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestAcceptConsentSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	redirect := "https://test.com/test"
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	consent.SetRequestedAccessTokenAudience([]string{"https://api.example.com"})
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo(redirect)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != consent.GetChallenge() {
				t.Fatalf("expected challenge string as %s, got %s", consent.GetChallenge(), *challenge)
			}

			// use reflect as cookie is a private attribute, also is a pointer so need to cast it multiple times
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())

			if !reflect.DeepEqual(acceptReq.GetGrantScope(), consent.GetRequestedScope()) {
				t.Fatalf("expected scope as %s, got %s", consent.GetRequestedScope(), acceptReq.GetGrantScope())
			}

			if !reflect.DeepEqual(acceptReq.GetGrantAccessTokenAudience(), consent.GetRequestedAccessTokenAudience()) {
				t.Fatalf("expected audience as %s, got %s", consent.GetRequestedAccessTokenAudience(), acceptReq.GetGrantAccessTokenAudience())
			}

			return accept, new(http.Response), nil
		},
	)

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")

	if a != accept {
		t.Fatalf("expected accept to be %v not  %v", accept, a)
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestAcceptConsentFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != consent.GetChallenge() {
				t.Fatalf("expected challenge string as %s, got %s", consent.GetChallenge(), *challenge)
			}

			// use reflect as cookie is a private attribute, also is a pointer so need to cast it multiple times
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())

			if !reflect.DeepEqual(acceptReq.GetGrantScope(), consent.GetRequestedScope()) {
				t.Fatalf("expected scope as %s, got %s", consent.GetRequestedScope(), acceptReq.GetGrantScope())
			}

			if !reflect.DeepEqual(acceptReq.GetGrantAccessTokenAudience(), consent.GetRequestedAccessTokenAudience()) {
				t.Fatalf("expected audience as %s, got %s", consent.GetRequestedAccessTokenAudience(), acceptReq.GetGrantAccessTokenAudience())
			}

			return nil, new(http.Response), fmt.Errorf("error")
		},
	)

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")

	if a != nil {
		t.Fatalf("expected accept to be nil not  %v", a)
	}

	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestAcceptConsentNoTenantID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	redirect := "https://test.com/test"
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	// No tenant_id in consent context
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo(redirect)

	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())
			sessionData := acceptReq.GetSession()
			// When no tenant_id is in the context, access_token should not have _tenant_id set
			accessTokenRaw := sessionData.GetAccessToken()
			if accessTokenRaw != nil {
				accessToken, _ := accessTokenRaw.(map[string]interface{})
				if _, ok := accessToken["_tenant_id"]; ok {
					t.Fatalf("expected no _tenant_id in access_token when consent context has none, got %v", accessToken)
				}
			}
			return accept, new(http.Response), nil
		},
	)

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")

	if a != accept {
		t.Fatalf("expected accept to be %v not %v", accept, a)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
}

func TestAcceptConsentPropagatesTenantIDFromContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	redirect := "https://test.com/test"
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	// tenant_id in context (primary path — set by AcceptLoginRequest)
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	consent.Context = map[string]interface{}{"tenant_id": "tenant-from-ctx"}
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo(redirect)

	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())
			sessionData := acceptReq.GetSession()
			accessTokenRaw := sessionData.GetAccessToken()
			accessToken, ok := accessTokenRaw.(map[string]interface{})
			if !ok {
				t.Fatalf("expected access_token to be map[string]interface{}, got %T", accessTokenRaw)
			}
			tenantID, ok := accessToken["_tenant_id"]
			if !ok {
				t.Fatalf("expected _tenant_id in session.access_token, got %v", accessToken)
			}
			if tenantID != "tenant-from-ctx" {
				t.Fatalf("expected _tenant_id=tenant-from-ctx, got %v", tenantID)
			}
			return accept, new(http.Response), nil
		},
	)

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "tenant-from-ctx")

	if a != accept {
		t.Fatalf("expected accept to be %v not %v", accept, a)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
}

func TestAcceptConsentGrantsResourceIndicators(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	consent.SetRequestedAccessTokenAudience([]string{"https://api.example.com"})
	client := hClient.NewOAuth2Client()
	client.SetAudience([]string{"https://api.example.com", "https://mcp.example.com"})
	consent.SetClient(*client)
	// "resource" appears four times: a new indicator below a registered audience,
	// a duplicate of the audience parameter, a relative URI, which RFC 8707
	// forbids, and a valid indicator the client is not registered for.
	consent.SetRequestUrl(
		"https://example.com/oauth2/auth?audience=https%3A%2F%2Fapi.example.com" +
			"&resource=https%3A%2F%2Fmcp.example.com%2Fsse" +
			"&resource=https%3A%2F%2Fapi.example.com" +
			"&resource=%2Fsse" +
			"&resource=https%3A%2F%2Fevil.example.com",
	)
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo("https://test.com/test")

	expectedAudience := []string{"https://api.example.com", "https://mcp.example.com/sse"}

	// one line for the malformed indicator, one for the unregistered one
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(2)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())

			if !reflect.DeepEqual(acceptReq.GetGrantAccessTokenAudience(), expectedAudience) {
				t.Fatalf("expected audience as %s, got %s", expectedAudience, acceptReq.GetGrantAccessTokenAudience())
			}

			return accept, new(http.Response), nil
		},
	)

	_, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")
	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
}

func TestAcceptConsentIgnoresResourcesWithoutClientAudience(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	// a client with no registered audience, as dynamic registration produces
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	consent.SetRequestedAccessTokenAudience([]string{"https://api.example.com"})
	consent.SetClient(*hClient.NewOAuth2Client())
	consent.SetRequestUrl("https://example.com/oauth2/auth?resource=https%3A%2F%2Fmcp.example.com%2Fsse")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo("https://test.com/test")

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())

			if !reflect.DeepEqual(acceptReq.GetGrantAccessTokenAudience(), consent.GetRequestedAccessTokenAudience()) {
				t.Fatalf("expected audience as %s, got %s", consent.GetRequestedAccessTokenAudience(), acceptReq.GetGrantAccessTokenAudience())
			}

			return accept, new(http.Response), nil
		},
	)

	_, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")
	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
}

func TestResourceIndicators(t *testing.T) {
	for _, test := range []struct {
		name           string
		requestURL     string
		valid, invalid []string
	}{
		{
			name:       "no resource parameter",
			requestURL: "https://example.com/oauth2/auth?client_id=test&scope=openid",
		},
		{
			name:       "multiple resources",
			requestURL: "https://example.com/oauth2/auth?resource=https%3A%2F%2Fa.example.com&resource=urn%3Aexample%3Aapi",
			valid:      []string{"https://a.example.com", "urn:example:api"},
		},
		{
			name:       "relative uri is rejected",
			requestURL: "https://example.com/oauth2/auth?resource=%2Fapi",
			invalid:    []string{"/api"},
		},
		{
			name:       "fragment is rejected",
			requestURL: "https://example.com/oauth2/auth?resource=https%3A%2F%2Fa.example.com%2Fapi%23frag",
			invalid:    []string{"https://a.example.com/api#frag"},
		},
		{
			name:       "empty resource is rejected",
			requestURL: "https://example.com/oauth2/auth?resource=",
			invalid:    []string{""},
		},
		{
			name:       "whitespace is rejected",
			requestURL: "https://example.com/oauth2/auth?resource=https%3A%2F%2Fa.example.com%2Fa+b",
			invalid:    []string{"https://a.example.com/a b"},
		},
		{
			name:       "valid and invalid resources are partitioned in order",
			requestURL: "https://example.com/oauth2/auth?resource=%2Fapi&resource=https%3A%2F%2Fa.example.com&resource=https%3A%2F%2Fb.example.com%23f&resource=urn%3Aexample%3Aapi",
			valid:      []string{"https://a.example.com", "urn:example:api"},
			invalid:    []string{"/api", "https://b.example.com#f"},
		},
		{
			name:       "malformed percent escape is dropped",
			requestURL: "https://example.com/oauth2/auth?resource=%%%&resource=https%3A%2F%2Fa.example.com",
			valid:      []string{"https://a.example.com"},
		},
		{
			name:       "unparsable request url",
			requestURL: "://bad?resource=https%3A%2F%2Fa.example.com",
		},
		{
			name:       "empty request url",
			requestURL: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid, invalid := resourceIndicators(test.requestURL)

			if !reflect.DeepEqual(valid, test.valid) {
				t.Fatalf("expected valid resources %v, got %v", test.valid, valid)
			}

			if !reflect.DeepEqual(invalid, test.invalid) {
				t.Fatalf("expected invalid resources %v, got %v", test.invalid, invalid)
			}
		})
	}
}

func TestMergeAudience(t *testing.T) {
	for _, test := range []struct {
		name      string
		audience  []string
		resources []string
		expected  []string
	}{
		{
			name:     "no resources keeps the audience as is",
			audience: []string{"https://api.example.com"},
			expected: []string{"https://api.example.com"},
		},
		{
			name:      "resources are appended to an empty audience",
			resources: []string{"https://mcp.example.com/sse"},
			expected:  []string{"https://mcp.example.com/sse"},
		},
		{
			name:      "repeated resources are granted once",
			resources: []string{"https://mcp.example.com/sse", "https://mcp.example.com/sse"},
			expected:  []string{"https://mcp.example.com/sse"},
		},
		{
			name:      "resource already in the audience is not repeated",
			audience:  []string{"https://api.example.com"},
			resources: []string{"https://api.example.com", "https://mcp.example.com/sse"},
			expected:  []string{"https://api.example.com", "https://mcp.example.com/sse"},
		},
		{
			name:      "audience case differences are distinct audiences",
			audience:  []string{"https://API.example.com"},
			resources: []string{"https://api.example.com"},
			expected:  []string{"https://API.example.com", "https://api.example.com"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// spare capacity catches a merge that writes into the caller's array
			audience := make([]string, len(test.audience), len(test.audience)+4)
			copy(audience, test.audience)

			merged := mergeAudience(audience, test.resources)

			if !reflect.DeepEqual(merged, test.expected) {
				t.Fatalf("expected audience %v, got %v", test.expected, merged)
			}

			if extra := audience[:cap(audience)][len(audience):]; !reflect.DeepEqual(extra, make([]string, len(extra))) {
				t.Fatalf("expected the caller's audience array to be untouched, got %v", extra)
			}
		})
	}
}

func TestAudiencePermits(t *testing.T) {
	for _, test := range []struct {
		name       string
		registered string
		resource   string
		permitted  bool
	}{
		{name: "exact match", registered: "https://api.example.com/v1", resource: "https://api.example.com/v1", permitted: true},
		{name: "path below the registered path", registered: "https://api.example.com/v1", resource: "https://api.example.com/v1/orders", permitted: true},
		{name: "path sharing only a string prefix", registered: "https://api.example.com/v1", resource: "https://api.example.com/v10"},
		{name: "resource with trailing slash", registered: "https://api.example.com/v1", resource: "https://api.example.com/v1/", permitted: true},
		{name: "registered with trailing slash", registered: "https://api.example.com/v1/", resource: "https://api.example.com/v1", permitted: true},
		{name: "query is ignored", registered: "https://api.example.com/v1", resource: "https://api.example.com/v1?x=1", permitted: true},
		{name: "registered host permits every path", registered: "https://api.example.com", resource: "https://api.example.com/anything/at/all", permitted: true},
		{name: "scheme mismatch", registered: "https://api.example.com", resource: "http://api.example.com"},
		{name: "host mismatch", registered: "https://api.example.com", resource: "https://evil.example.com"},
		{name: "port mismatch", registered: "https://api.example.com", resource: "https://api.example.com:8443"},
		{name: "same opaque uri", registered: "urn:example:a", resource: "urn:example:a", permitted: true},
		{name: "different opaque uri", registered: "urn:example:a", resource: "urn:example:b"},
		{name: "unparsable registered audience", registered: "://bad", resource: "https://api.example.com"},
		{name: "unparsable resource", registered: "https://api.example.com", resource: "://bad"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if permitted := audiencePermits(test.registered, test.resource); permitted != test.permitted {
				t.Fatalf("expected %q to permit %q: %v, got %v", test.registered, test.resource, test.permitted, permitted)
			}
		})
	}
}

func TestPermittedResources(t *testing.T) {
	for _, test := range []struct {
		name                string
		registered          []string
		resources           []string
		permitted, rejected []string
	}{
		{
			name:      "empty registered audience rejects everything",
			resources: []string{"https://a.example.com", "https://b.example.com"},
			rejected:  []string{"https://a.example.com", "https://b.example.com"},
		},
		{
			name:       "no resources",
			registered: []string{"https://a.example.com"},
		},
		{
			name:       "partitioned in order against any registered entry",
			registered: []string{"https://a.example.com", "urn:example:api"},
			resources:  []string{"https://b.example.com", "urn:example:api", "https://a.example.com/v1", "urn:example:other"},
			permitted:  []string{"urn:example:api", "https://a.example.com/v1"},
			rejected:   []string{"https://b.example.com", "urn:example:other"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			permitted, rejected := permittedResources(test.registered, test.resources)

			if !reflect.DeepEqual(permitted, test.permitted) {
				t.Fatalf("expected permitted resources %v, got %v", test.permitted, permitted)
			}

			if !reflect.DeepEqual(rejected, test.rejected) {
				t.Fatalf("expected rejected resources %v, got %v", test.rejected, rejected)
			}
		})
	}
}
