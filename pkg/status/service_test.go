package status

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/golang/mock/gomock"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_status.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_tracing.go go.opentelemetry.io/otel/trace Tracer
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_kratos.go github.com/ory/kratos-client-go MetadataApi
//go:generate mockgen -build_flags=--mod=mod -package status -destination ./mock_hydra.go -mock_names MetadataApi=MockHydraMetadataApi "github.com/ory/hydra-client-go/v2" MetadataApi

func TestCheckKratosReadySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosMetadataApi := NewMockMetadataApi(ctrl)
	mockHydraMetadataApi := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	isReadyReturn := kClient.MetadataApiIsReadyRequest{
		ApiService: mockKratosMetadataApi,
	}

	mockTracer.EXPECT().Start(ctx, "status.Service.CheckKratosReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(1.0))
	mockKratosMetadataApi.EXPECT().IsReady(ctx).Times(1).Return(isReadyReturn)

	mockKratosMetadataApi.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.MetadataApiIsReadyRequest) (*kClient.IsAlive200Response, *http.Response, error) {
			isAlive := kClient.NewIsAlive200ResponseWithDefaults()
			httpResp := new(http.Response)
			httpResp.StatusCode = 200
			return isAlive, httpResp, nil
		},
	)

	r, _ := NewService(mockKratosMetadataApi, mockHydraMetadataApi, mockTracer, mockMonitor, mockLogger).CheckKratosReady(ctx)

	if !r {
		t.Fatalf("expected response to be %v not  %v", true, false)
	}
}

func TestCheckHydraReadySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosMetadataApi := NewMockMetadataApi(ctrl)
	mockHydraMetadataApi := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	isReadyReturn := hClient.MetadataApiIsReadyRequest{
		ApiService: mockHydraMetadataApi,
	}

	mockTracer.EXPECT().Start(ctx, "status.Service.CheckHydraReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(1.0))
	mockHydraMetadataApi.EXPECT().IsReady(gomock.Any()).Times(1).Return(isReadyReturn)

	mockHydraMetadataApi.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.MetadataApiIsReadyRequest) (*hClient.IsReady200Response, *http.Response, error) {
			isReady := hClient.NewIsReady200ResponseWithDefaults()
			httpResp := new(http.Response)
			httpResp.StatusCode = 200
			return isReady, httpResp, nil
		},
	)

	r, _ := NewService(mockKratosMetadataApi, mockHydraMetadataApi, mockTracer, mockMonitor, mockLogger).CheckHydraReady(ctx)

	if !r {
		t.Fatalf("expected response to be %v not  %v", true, false)
	}
}

func TestCheckKratosReadyFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosMetadataApi := NewMockMetadataApi(ctrl)
	mockHydraMetadataApi := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	isReadyReturn := kClient.MetadataApiIsReadyRequest{
		ApiService: mockKratosMetadataApi,
	}

	mockTracer.EXPECT().Start(ctx, "status.Service.CheckKratosReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(0.0))
	mockKratosMetadataApi.EXPECT().IsReady(ctx).Times(1).Return(isReadyReturn)

	mockKratosMetadataApi.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.MetadataApiIsReadyRequest) (*kClient.IsAlive200Response, *http.Response, error) {
			httpResp := new(http.Response)
			httpResp.StatusCode = 500
			return nil, httpResp, errors.New("Test Error")
		},
	)

	r, err := NewService(mockKratosMetadataApi, mockHydraMetadataApi, mockTracer, mockMonitor, mockLogger).CheckKratosReady(ctx)

	if r {
		t.Fatalf("expected response to be %v not  %v", false, true)
	}

	if err == nil {
		t.Fatal("expected error to not be nil")
	}
}

func TestCheckHydraReadyFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosMetadataApi := NewMockMetadataApi(ctrl)
	mockHydraMetadataApi := NewMockHydraMetadataApi(ctrl)

	ctx := context.Background()

	isReadyReturn := hClient.MetadataApiIsReadyRequest{
		ApiService: mockHydraMetadataApi,
	}

	mockTracer.EXPECT().Start(ctx, "status.Service.CheckHydraReady").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockMonitor.EXPECT().SetDependencyAvailability(gomock.Any(), float64(0.0))
	mockHydraMetadataApi.EXPECT().IsReady(gomock.Any()).Times(1).Return(isReadyReturn)

	mockHydraMetadataApi.EXPECT().IsReadyExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.MetadataApiIsReadyRequest) (*hClient.IsReady200Response, *http.Response, error) {
			httpResp := new(http.Response)
			httpResp.StatusCode = 500
			return nil, httpResp, errors.New("Test Error")
		},
	)

	r, err := NewService(mockKratosMetadataApi, mockHydraMetadataApi, mockTracer, mockMonitor, mockLogger).CheckHydraReady(ctx)

	if r {
		t.Fatalf("expected response to be %v not  %v", false, true)
	}

	if err == nil {
		t.Fatal("expected error to not be nil")
	}
}
