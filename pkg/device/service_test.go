package device

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	reflect "reflect"
	"testing"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	hClient "github.com/ory/hydra-client-go/v2"
	trace "go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_device.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_hydra.go -source=../../internal/hydra/interfaces.go

func TestParseUserCodeBodySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	code := "ABCDEFGH"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).ParseUserCodeBody(req)

	actual, _ := b.MarshalJSON()
	if !reflect.DeepEqual(actual, jsonBody) {
		t.Fatalf("expected flow to be %v not %v", jsonBody, actual)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestParseUserCodeBodyFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	body := []byte("aaaaa")

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(body)))

	_, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).ParseUserCodeBody(req)

	if err == nil {
		t.Fatal("expected error to be not nil")
	}
}

func TestAcceptUserCodeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockAuth2Api := NewMockOAuth2Api(ctrl)

	ctx := context.Background()

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	redirectTo := hClient.NewOAuth2RedirectTo("test")

	req := hydra.ApiAcceptUserCodeRequestRequest{
		ApiService: mockAuth2Api,
	}
	resp := http.Response{}

	mockTracer.EXPECT().Start(ctx, "device.service.AcceptUserCode").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockAuth2Api)
	mockAuth2Api.EXPECT().AcceptUserCodeRequest(ctx).Times(1).Return(req)
	mockAuth2Api.EXPECT().AcceptUserCodeRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hydra.ApiAcceptUserCodeRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			if _challenge := (*string)(reflect.ValueOf(r).FieldByName("deviceChallenge").UnsafePointer()); *_challenge != challenge {
				t.Fatalf("expected challenge to be %s, got %s", challenge, *_challenge)
			}

			return redirectTo, &resp, nil
		},
	)

	rt, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptUserCode(ctx, challenge, userCodeRequest)

	if rt != redirectTo {
		t.Fatalf("expected redirect to be %v not  %v", redirectTo, rt)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestAcceptUserCodeFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockAuth2Api := NewMockOAuth2Api(ctrl)

	ctx := context.Background()

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hydra.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code

	req := hydra.ApiAcceptUserCodeRequestRequest{
		ApiService: mockAuth2Api,
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "device.service.AcceptUserCode").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockAuth2Api)
	mockAuth2Api.EXPECT().AcceptUserCodeRequest(ctx).Times(1).Return(req)
	mockAuth2Api.EXPECT().AcceptUserCodeRequestExecute(gomock.Any()).Times(1).Return(nil, nil, fmt.Errorf("error"))

	rt, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptUserCode(ctx, challenge, userCodeRequest)

	if rt != nil {
		t.Fatalf("expected redirect to be nil not  %v", rt)
	}
	if err == nil {
		t.Fatalf("expected error to be not nil")
	}
}
