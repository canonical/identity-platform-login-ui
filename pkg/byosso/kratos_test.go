// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package byosso -destination ./mock_kratos.go github.com/ory/kratos-client-go/v25 FrontendAPI
//go:generate mockgen -build_flags=--mod=mod -package byosso -destination ./mock_identity.go github.com/ory/kratos-client-go/v25 IdentityAPI

func TestVerificationStartSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := NewMockLoggerInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockKratosFrontendAPI := NewMockFrontendAPI(ctrl)

	ctx := context.Background()
	returnTo := "https://example.com/ui/login?login_challenge=test"
	flow := kClient.NewVerificationFlowWithDefaults()
	flow.Id = "test"
	flow.Ui.Nodes = []kClient.UiNode{inputNode("default", "csrf_token", "csrf")}

	mockTracer.EXPECT().Start(ctx, "byosso.Verification.Start").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(2).Return(mockKratosFrontendAPI)
	mockKratosFrontendAPI.EXPECT().CreateBrowserVerificationFlow(ctx).Times(1).Return(kClient.FrontendAPICreateBrowserVerificationFlowRequest{ApiService: mockKratosFrontendAPI})
	mockKratosFrontendAPI.EXPECT().CreateBrowserVerificationFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendAPICreateBrowserVerificationFlowRequest) (*kClient.VerificationFlow, *http.Response, error) {
			// use reflect as returnTo is a private attribute, also is a string pointer so need to cast it multiple times
			if rt := (*string)(reflect.ValueOf(r).FieldByName("returnTo").UnsafePointer()); *rt != returnTo {
				t.Fatalf("expected returnTo to be %s, got %s", returnTo, *rt)
			}
			return flow, &http.Response{Header: http.Header{"Set-Cookie": []string{"csrf_token=csrf"}}}, nil
		},
	)
	mockKratosFrontendAPI.EXPECT().UpdateVerificationFlow(ctx).Times(1).Return(kClient.FrontendAPIUpdateVerificationFlowRequest{ApiService: mockKratosFrontendAPI})
	mockKratosFrontendAPI.EXPECT().UpdateVerificationFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendAPIUpdateVerificationFlowRequest) (*kClient.VerificationFlow, *http.Response, error) {
			if id := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *id != flow.Id {
				t.Fatalf("expected flow id to be %s, got %s", flow.Id, *id)
			}
			// the cookies of the browser and the ones Kratos set with the flow
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "browser=test; csrf_token=csrf" {
				t.Fatalf("expected cookie to be browser=test; csrf_token=csrf, got %s", *cookie)
			}
			body := (*kClient.UpdateVerificationFlowBody)(reflect.ValueOf(r).FieldByName("updateVerificationFlowBody").UnsafePointer())
			code := body.UpdateVerificationFlowWithCodeMethod
			if code == nil {
				t.Fatalf("expected the code method")
			}
			if code.GetEmail() != "test@example.com" {
				t.Fatalf("expected email to be test@example.com, got %s", code.GetEmail())
			}
			if code.GetCsrfToken() != "csrf" {
				t.Fatalf("expected csrf token to be csrf, got %s", code.GetCsrfToken())
			}
			return flow, &http.Response{Header: http.Header{"Set-Cookie": []string{"continuity=test"}}}, nil
		},
	)

	flowID, flowCookies, err := NewVerification(mockKratos, mockTracer, mockLogger).Start(ctx, returnTo, "test@example.com", []*http.Cookie{{Name: "browser", Value: "test"}})

	expectNoError(t, err)
	if flowID != flow.Id {
		t.Fatalf("expected flow id to be %s, got %s", flow.Id, flowID)
	}
	if len(flowCookies) != 2 || flowCookies[0].Name != "csrf_token" || flowCookies[1].Name != "continuity" {
		t.Fatalf("expected cookies csrf_token and continuity, got %v", flowCookies)
	}
}

func TestIdentityFinderSuccess(t *testing.T) {
	ctx := context.Background()
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"email": "test@example.com"})

	// list expects one lookup of the lower-cased, trimmed email, answered with identities
	list := func(t *testing.T, api *MockIdentityAPI, identities ...kClient.Identity) {
		api.EXPECT().ListIdentities(ctx).Times(1).Return(kClient.IdentityAPIListIdentitiesRequest{ApiService: api})
		api.EXPECT().ListIdentitiesExecute(gomock.Any()).Times(1).DoAndReturn(
			func(r kClient.IdentityAPIListIdentitiesRequest) ([]kClient.Identity, *http.Response, error) {
				// use reflect as credentialsIdentifier is a private attribute, also is a string pointer so need to cast it multiple times
				if identifier := (*string)(reflect.ValueOf(r).FieldByName("credentialsIdentifier").UnsafePointer()); *identifier != "test@example.com" {
					t.Fatalf("expected the lower-cased, trimmed email as identifier, got %s", *identifier)
				}
				return identities, new(http.Response), nil
			},
		)
	}
	// get expects the identity to be read with its recovery codes, and answers it with credentials
	get := func(t *testing.T, api *MockIdentityAPI, credentials map[string]kClient.IdentityCredentials) {
		withCredentials := *identity
		withCredentials.SetCredentials(credentials)
		api.EXPECT().GetIdentity(ctx, "test").Times(1).Return(kClient.IdentityAPIGetIdentityRequest{ApiService: api})
		api.EXPECT().GetIdentityExecute(gomock.Any()).Times(1).DoAndReturn(
			func(r kClient.IdentityAPIGetIdentityRequest) (*kClient.Identity, *http.Response, error) {
				// use reflect as includeCredential is a private attribute, also is a pointer so need to cast it multiple times
				if include := (*[]string)(reflect.ValueOf(r).FieldByName("includeCredential").UnsafePointer()); !reflect.DeepEqual(*include, []string{"lookup_secret"}) {
					t.Fatalf("expected the lookup_secret credential to be included, got %v", *include)
				}
				return &withCredentials, new(http.Response), nil
			},
		)
	}

	// span is empty when Kratos is not called
	tests := []struct {
		name     string
		span     string
		call     func(t *testing.T, api *MockIdentityAPI, f *IdentityFinder) (any, error)
		expected any
	}{
		{name: "IdentityID", span: "kratos.IdentityAPI.ListIdentities", expected: "test", call: func(t *testing.T, api *MockIdentityAPI, f *IdentityFinder) (any, error) {
			list(t, api, *identity)
			return f.IdentityID(ctx, " Test@Example.com ")
		}},
		{name: "IdentityID with an empty email", expected: "", call: func(_ *testing.T, _ *MockIdentityAPI, f *IdentityFinder) (any, error) {
			return f.IdentityID(ctx, " ")
		}},
		{name: "IdentityExists", span: "kratos.IdentityAPI.ListIdentities", expected: true, call: func(t *testing.T, api *MockIdentityAPI, f *IdentityFinder) (any, error) {
			list(t, api, *identity)
			return f.IdentityExists(ctx, "test@example.com")
		}},
		{name: "IdentityExists without identity", span: "kratos.IdentityAPI.ListIdentities", expected: false, call: func(t *testing.T, api *MockIdentityAPI, f *IdentityFinder) (any, error) {
			list(t, api)
			return f.IdentityExists(ctx, "test@example.com")
		}},
		{name: "HasRecoveryCodes", span: "kratos.IdentityAPI.GetIdentity", expected: true, call: func(t *testing.T, api *MockIdentityAPI, f *IdentityFinder) (any, error) {
			get(t, api, map[string]kClient.IdentityCredentials{"lookup_secret": *kClient.NewIdentityCredentials()})
			return f.HasRecoveryCodes(ctx, "test")
		}},
		{name: "HasRecoveryCodes without recovery codes", span: "kratos.IdentityAPI.GetIdentity", expected: false, call: func(t *testing.T, api *MockIdentityAPI, f *IdentityFinder) (any, error) {
			get(t, api, map[string]kClient.IdentityCredentials{})
			return f.HasRecoveryCodes(ctx, "test")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := NewMockLoggerInterface(ctrl)
			mockKratosAdmin := NewMockKratosAdminClientInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockIdentityAPI := NewMockIdentityAPI(ctrl)

			if tt.span != "" {
				mockTracer.EXPECT().Start(ctx, tt.span).Times(1).Return(ctx, trace.SpanFromContext(ctx))
				mockKratosAdmin.EXPECT().IdentityApi().Times(1).Return(mockIdentityAPI)
			}

			got, err := tt.call(t, mockIdentityAPI, NewIdentityFinder(mockKratosAdmin, mockTracer, mockLogger))

			expectNoError(t, err)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
